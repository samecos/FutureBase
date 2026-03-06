package com.archplatform.project.repository;

import com.archplatform.project.entity.Project;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface ProjectRepository extends JpaRepository<Project, UUID> {

    Optional<Project> findBySlug(String slug);

    @Query("SELECT p FROM Project p WHERE p.id = :id AND p.deletedAt IS NULL")
    Optional<Project> findActiveById(@Param("id") UUID id);

    @Query("SELECT p FROM Project p WHERE p.slug = :slug AND p.deletedAt IS NULL")
    Optional<Project> findActiveBySlug(@Param("slug") String slug);

    @Query("SELECT p FROM Project p WHERE p.tenantId = :tenantId AND p.deletedAt IS NULL ORDER BY p.updatedAt DESC")
    List<Project> findAllByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT p FROM Project p WHERE p.tenantId = :tenantId AND p.deletedAt IS NULL ORDER BY p.updatedAt DESC")
    Page<Project> findAllByTenantId(@Param("tenantId") UUID tenantId, Pageable pageable);

    @Query("SELECT p FROM Project p WHERE p.ownerId = :ownerId AND p.deletedAt IS NULL ORDER BY p.updatedAt DESC")
    List<Project> findAllByOwnerId(@Param("ownerId") UUID ownerId);

    @Query("SELECT p FROM Project p WHERE p.tenantId = :tenantId AND p.status = :status AND p.deletedAt IS NULL")
    List<Project> findAllByTenantIdAndStatus(@Param("tenantId") UUID tenantId, @Param("status") Project.ProjectStatus status);

    @Query("SELECT p FROM Project p WHERE p.tenantId = :tenantId AND p.visibility = :visibility AND p.deletedAt IS NULL")
    List<Project> findAllByTenantIdAndVisibility(@Param("tenantId") UUID tenantId, @Param("visibility") Project.ProjectVisibility visibility);

    @Query("SELECT COUNT(p) FROM Project p WHERE p.tenantId = :tenantId AND p.deletedAt IS NULL")
    long countByTenantId(@Param("tenantId") UUID tenantId);

    @Query("SELECT COUNT(p) FROM Project p WHERE p.ownerId = :ownerId AND p.deletedAt IS NULL")
    long countByOwnerId(@Param("ownerId") UUID ownerId);

    @Query("SELECT p FROM Project p WHERE p.name ILIKE %:keyword% OR p.description ILIKE %:keyword% AND p.tenantId = :tenantId AND p.deletedAt IS NULL")
    List<Project> searchByKeyword(@Param("tenantId") UUID tenantId, @Param("keyword") String keyword);

    @Modifying
    @Query("UPDATE Project p SET p.designCount = p.designCount + 1 WHERE p.id = :projectId")
    void incrementDesignCount(@Param("projectId") UUID projectId);

    @Modifying
    @Query("UPDATE Project p SET p.designCount = p.designCount - 1 WHERE p.id = :projectId AND p.designCount > 0")
    void decrementDesignCount(@Param("projectId") UUID projectId);

    @Modifying
    @Query("UPDATE Project p SET p.memberCount = p.memberCount + 1 WHERE p.id = :projectId")
    void incrementMemberCount(@Param("projectId") UUID projectId);

    @Modifying
    @Query("UPDATE Project p SET p.memberCount = p.memberCount - 1 WHERE p.id = :projectId AND p.memberCount > 0")
    void decrementMemberCount(@Param("projectId") UUID projectId);

    @Modifying
    @Query("UPDATE Project p SET p.totalStorageBytes = p.totalStorageBytes + :bytes WHERE p.id = :projectId")
    void addStorageBytes(@Param("projectId") UUID projectId, @Param("bytes") long bytes);

    @Modifying
    @Query("UPDATE Project p SET p.totalStorageBytes = p.totalStorageBytes - :bytes WHERE p.id = :projectId AND p.totalStorageBytes >= :bytes")
    void subtractStorageBytes(@Param("projectId") UUID projectId, @Param("bytes") long bytes);

    @Modifying
    @Query("UPDATE Project p SET p.status = :status, p.archivedAt = :archivedAt WHERE p.id = :projectId")
    void archiveProject(@Param("projectId") UUID projectId, @Param("status") Project.ProjectStatus status, @Param("archivedAt") LocalDateTime archivedAt);

    @Modifying
    @Query("UPDATE Project p SET p.deletedAt = :deletedAt WHERE p.id = :projectId")
    void softDelete(@Param("projectId") UUID projectId, @Param("deletedAt") LocalDateTime deletedAt);

    @Modifying
    @Query("UPDATE Project p SET p.status = :status, p.completedAt = :completedAt WHERE p.id = :projectId")
    void markAsCompleted(@Param("projectId") UUID projectId, @Param("status") Project.ProjectStatus status, @Param("completedAt") LocalDateTime completedAt);

    boolean existsBySlugAndTenantId(String slug, UUID tenantId);

    @Query("SELECT p FROM Project p JOIN p.members m WHERE m.userId = :userId AND p.deletedAt IS NULL ORDER BY p.updatedAt DESC")
    List<Project> findAllByMemberUserId(@Param("userId") UUID userId);

    @Query("SELECT p FROM Project p JOIN p.members m WHERE m.userId = :userId AND p.deletedAt IS NULL")
    Page<Project> findAllByMemberUserId(@Param("userId") UUID userId, Pageable pageable);
}
