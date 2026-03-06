package com.archplatform.project.repository;

import com.archplatform.project.entity.ProjectMember;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface ProjectMemberRepository extends JpaRepository<ProjectMember, UUID> {

    Optional<ProjectMember> findByProjectIdAndUserId(UUID projectId, UUID userId);

    @Query("SELECT pm FROM ProjectMember pm WHERE pm.project.id = :projectId ORDER BY pm.role DESC, pm.createdAt ASC")
    List<ProjectMember> findAllByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT pm FROM ProjectMember pm WHERE pm.userId = :userId ORDER BY pm.lastAccessedAt DESC")
    List<ProjectMember> findAllByUserId(@Param("userId") UUID userId);

    boolean existsByProjectIdAndUserId(UUID projectId, UUID userId);

    @Query("SELECT COUNT(pm) FROM ProjectMember pm WHERE pm.project.id = :projectId")
    long countByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT pm FROM ProjectMember pm WHERE pm.project.id = :projectId AND pm.role = :role")
    List<ProjectMember> findAllByProjectIdAndRole(@Param("projectId") UUID projectId, @Param("role") ProjectMember.MemberRole role);

    @Modifying
    @Query("UPDATE ProjectMember pm SET pm.role = :role WHERE pm.id = :memberId")
    void updateRole(@Param("memberId") UUID memberId, @Param("role") ProjectMember.MemberRole role);

    @Modifying
    @Query("UPDATE ProjectMember pm SET pm.lastAccessedAt = CURRENT_TIMESTAMP WHERE pm.project.id = :projectId AND pm.userId = :userId")
    void updateLastAccessed(@Param("projectId") UUID projectId, @Param("userId") UUID userId);

    @Query("SELECT pm FROM ProjectMember pm WHERE pm.project.id = :projectId AND pm.role IN ('OWNER', 'ADMIN')")
    List<ProjectMember> findAdminsByProjectId(@Param("projectId") UUID projectId);
}
