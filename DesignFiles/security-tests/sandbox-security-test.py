#!/usr/bin/env python3
"""
半自动化建筑设计平台 - 沙箱安全测试脚本
用途: 验证Python脚本执行沙箱的隔离性和安全性
"""

import unittest
import json
import requests
import subprocess
import time
from typing import Dict, Any, Optional


class SandboxSecurityTest(unittest.TestCase):
    """沙箱安全测试套件"""
    
    @classmethod
    def setUpClass(cls):
        """测试类初始化"""
        cls.base_url = "http://localhost:8080"
        cls.api_version = "v1"
        cls.test_token = "test-security-token"
        cls.headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {cls.test_token}"
        }
        
        # 验证服务可用性
        try:
            response = requests.get(f"{cls.base_url}/health", timeout=5)
            if response.status_code != 200:
                raise Exception("服务健康检查失败")
        except Exception as e:
            raise Exception(f"无法连接到测试服务: {e}")
    
    def execute_script(self, script: str, timeout: int = 5000) -> Dict[str, Any]:
        """执行脚本并返回结果"""
        payload = {
            "script": script,
            "timeout": timeout
        }
        
        response = requests.post(
            f"{self.base_url}/api/{self.api_version}/scripts/execute",
            headers=self.headers,
            json=payload,
            timeout=30
        )
        
        return {
            "status_code": response.status_code,
            "response": response.json() if response.status_code == 200 else None,
            "text": response.text
        }
    
    # ========== 文件系统隔离测试 ==========
    
    def test_001_file_system_read_blocked(self):
        """测试1: 文件系统读取应该被阻止"""
        script = """
import os
try:
    with open('/etc/passwd', 'r') as f:
        content = f.read()
    result = {"success": True, "content_length": len(content)}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        # 应该返回失败结果
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "文件系统读取应该被阻止")
        self.assertIn("error", response_data.get("result", {}),
                     "应该返回错误信息")
        print("✓ 文件系统读取隔离测试通过")
    
    def test_002_file_system_write_blocked(self):
        """测试2: 文件系统写入应该被阻止"""
        script = """
import os
try:
    with open('/tmp/sandbox_test.txt', 'w') as f:
        f.write('test content')
    result = {"success": True}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "文件系统写入应该被阻止")
        print("✓ 文件系统写入隔离测试通过")
    
    def test_003_directory_listing_blocked(self):
        """测试3: 目录列表应该被阻止"""
        script = """
import os
try:
    files = os.listdir('/etc')
    result = {"success": True, "files": files}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "目录列表应该被阻止")
        print("✓ 目录列表隔离测试通过")
    
    # ========== 网络访问隔离测试 ==========
    
    def test_004_http_request_blocked(self):
        """测试4: HTTP请求应该被阻止"""
        script = """
try:
    import urllib.request
    response = urllib.request.urlopen('http://example.com', timeout=5)
    content = response.read()
    result = {"success": True, "content_length": len(content)}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "HTTP请求应该被阻止")
        print("✓ HTTP请求隔离测试通过")
    
    def test_005_socket_connection_blocked(self):
        """测试5: Socket连接应该被阻止"""
        script = """
import socket
try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(5)
    s.connect(('8.8.8.8', 53))
    result = {"success": True}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "Socket连接应该被阻止")
        print("✓ Socket连接隔离测试通过")
    
    def test_006_import_blocked_modules(self):
        """测试6: 危险模块导入应该被阻止"""
        dangerous_modules = [
            'subprocess',
            'os.system',
            'socket',
            'urllib.request',
        ]
        
        for module in dangerous_modules:
            script = f"""
try:
    import {module}
    result = {{"success": True}}
except Exception as e:
    result = {{"success": False, "error": str(e)}}
"""
            result = self.execute_script(script)
            
            self.assertEqual(result["status_code"], 200)
            response_data = result["response"]
            
            # 某些模块可能允许导入但功能受限
            print(f"  模块 {module}: {'允许导入但受限' if response_data.get('result', {}).get('success') else '导入被阻止'}")
        
        print("✓ 危险模块导入测试完成")
    
    # ========== 资源限制测试 ==========
    
    def test_007_memory_limit_enforced(self):
        """测试7: 内存限制应该生效"""
        script = """
try:
    # 尝试分配大量内存 (尝试分配500MB)
    big_list = []
    for i in range(50):
        big_list.append("x" * (10 * 1024 * 1024))  # 10MB per string
    result = {"success": True, "allocated": len(big_list)}
except MemoryError as e:
    result = {"success": False, "error": "Memory limit exceeded", "type": "MemoryError"}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script, timeout=10000)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        # 应该因为内存限制而失败
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "内存限制应该生效")
        print("✓ 内存限制测试通过")
    
    def test_008_cpu_time_limit_enforced(self):
        """测试8: CPU时间限制应该生效"""
        script = """
import time
start_time = time.time()
try:
    # CPU密集型操作
    total = 0
    for i in range(100000000):
        total += i
    result = {"success": True, "total": total}
except Exception as e:
    result = {"success": False, "error": str(e), "elapsed": time.time() - start_time}
"""
        result = self.execute_script(script, timeout=3000)  # 3秒超时
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        # 应该因为超时而失败
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "CPU时间限制应该生效")
        print("✓ CPU时间限制测试通过")
    
    def test_009_infinite_loop_blocked(self):
        """测试9: 无限循环应该被阻止"""
        script = """
import time
start_time = time.time()
try:
    while True:
        pass
except Exception as e:
    result = {"success": False, "error": str(e), "elapsed": time.time() - start_time}
"""
        result = self.execute_script(script, timeout=2000)  # 2秒超时
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "无限循环应该被阻止")
        print("✓ 无限循环阻止测试通过")
    
    # ========== 代码注入防护测试 ==========
    
    def test_010_eval_blocked(self):
        """测试10: eval应该被阻止或受限"""
        script = """
try:
    result = eval("__import__('os').system('ls')")
    result = {"success": True, "result": result}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        # eval应该被阻止或返回受限结果
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "eval应该被阻止")
        print("✓ eval阻止测试通过")
    
    def test_011_exec_blocked(self):
        """测试11: exec应该被阻止或受限"""
        script = """
try:
    exec("import os; os.system('ls')")
    result = {"success": True}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "exec应该被阻止")
        print("✓ exec阻止测试通过")
    
    def test_012_compile_blocked(self):
        """测试12: compile应该被阻止或受限"""
        script = """
try:
    code = compile("import os; os.system('ls')", "<string>", "exec")
    exec(code)
    result = {"success": True}
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertFalse(response_data.get("result", {}).get("success", True),
                        "compile应该被阻止")
        print("✓ compile阻止测试通过")
    
    # ========== 正常功能测试 ==========
    
    def test_013_normal_script_execution(self):
        """测试13: 正常脚本应该可以执行"""
        script = """
import math

def calculate_circle_area(radius):
    return math.pi * radius ** 2

areas = []
for r in range(1, 6):
    areas.append(calculate_circle_area(r))

result = {
    "success": True,
    "areas": areas,
    "count": len(areas)
}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        response_data = result["response"]
        
        self.assertTrue(response_data.get("result", {}).get("success", False),
                       "正常脚本应该执行成功")
        self.assertEqual(len(response_data.get("result", {}).get("areas", [])), 5,
                        "应该返回5个面积值")
        print("✓ 正常脚本执行测试通过")
    
    def test_014_api_access_allowed(self):
        """测试14: 平台API应该可以访问"""
        script = """
# 模拟访问平台API
from platform_api import Geometry

try:
    point = Geometry.create_point(x=10, y=20)
    result = {
        "success": True,
        "point": {"x": point.x, "y": point.y}
    }
except Exception as e:
    result = {"success": False, "error": str(e)}
"""
        result = self.execute_script(script)
        
        self.assertEqual(result["status_code"], 200)
        # 注意: 这个测试取决于平台API的实际实现
        print("✓ 平台API访问测试完成")


class AuthenticationSecurityTest(unittest.TestCase):
    """认证安全测试套件"""
    
    @classmethod
    def setUpClass(cls):
        cls.base_url = "http://localhost:8080"
        cls.api_version = "v1"
    
    def test_001_invalid_token_rejected(self):
        """测试1: 无效Token应该被拒绝"""
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer invalid-token"
        }
        
        response = requests.get(
            f"{self.base_url}/api/{self.api_version}/documents",
            headers=headers,
            timeout=10
        )
        
        self.assertEqual(response.status_code, 401,
                        "无效Token应该返回401")
        print("✓ 无效Token拒绝测试通过")
    
    def test_002_missing_token_rejected(self):
        """测试2: 缺少Token应该被拒绝"""
        headers = {
            "Content-Type": "application/json"
        }
        
        response = requests.get(
            f"{self.base_url}/api/{self.api_version}/documents",
            headers=headers,
            timeout=10
        )
        
        self.assertEqual(response.status_code, 401,
                        "缺少Token应该返回401")
        print("✓ 缺少Token拒绝测试通过")
    
    def test_003_sql_injection_prevention(self):
        """测试3: SQL注入应该被阻止"""
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer test-token"
        }
        
        # 尝试SQL注入
        malicious_id = "1' OR '1'='1"
        
        response = requests.get(
            f"{self.base_url}/api/{self.api_version}/documents/{malicious_id}",
            headers=headers,
            timeout=10
        )
        
        # 不应该返回正常数据
        self.assertIn(response.status_code, [400, 404, 422],
                     "SQL注入应该被阻止")
        print("✓ SQL注入防护测试通过")


def run_all_tests():
    """运行所有安全测试"""
    print("=" * 60)
    print("半自动化建筑设计平台 - 沙箱安全测试")
    print("=" * 60)
    print()
    
    # 创建测试套件
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()
    
    # 添加测试类
    suite.addTests(loader.loadTestsFromTestCase(SandboxSecurityTest))
    suite.addTests(loader.loadTestsFromTestCase(AuthenticationSecurityTest))
    
    # 运行测试
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    
    print()
    print("=" * 60)
    print("测试结果汇总")
    print("=" * 60)
    print(f"总测试数: {result.testsRun}")
    print(f"成功: {result.testsRun - len(result.failures) - len(result.errors)}")
    print(f"失败: {len(result.failures)}")
    print(f"错误: {len(result.errors)}")
    print()
    
    if result.wasSuccessful():
        print("✓ 所有安全测试通过!")
    else:
        print("✗ 部分测试未通过，请检查安全实现")
    
    return result.wasSuccessful()


if __name__ == '__main__':
    success = run_all_tests()
    exit(0 if success else 1)
