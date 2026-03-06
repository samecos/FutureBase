package com.archplatform.user.repository;

import com.archplatform.user.entity.Role;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface RoleRepository extends JpaRepository<Role, UUID> {

    Optional<Role> findByName(String name);

    @Query("SELECT r FROM Role r WHERE r.name = :name AND r.tenantId IS NULL")
    Optional<Role> findSystemRoleByName(@Param("name") String name);

    @Query("SELECT r FROM Role r WHERE r.name = :name AND r.tenantId = :tenantId")
    Optional<Role> findByNameAndTenantId(@Param("name") String name, @Param("tenantId") UUID tenantId);

    @Query("SELECT r FROM Role r WHERE r.tenantId IS NULL")
    List<Role> findAllSystemRoles();

    @Query("SELECT r FROM Role r WHERE r.tenantId = :tenantId")
    List<Role> findAllByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT r FROM Role r WHERE r.tenantId IS NULL OR r.tenantId = :tenantId")
    List<Role> findAllAvailableRoles(@Param("tenantId") UUID tenantId);

    @Query("SELECT r FROM Role r JOIN r.users u WHERE u.id = :userId")
    List<Role> findAllByUserId(@Param("userId") UUID userId);

    boolean existsByNameAndTenantId(String name, UUID tenantId);

    @Query("SELECT r FROM Role r WHERE r.level = :level")
    List<Role> findAllByLevel(@Param("level") Role.RoleLevel level);
}
