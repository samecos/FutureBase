package com.archplatform.project.repository;

import com.archplatform.project.entity.Design;
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
public interface DesignRepository extends JpaRepository<Design, UUID> {

    @Query("SELECT d FROM Design d WHERE d.id = :id AND d.deletedAt IS NULL")
    Optional<Design> findActiveById(@Param("id") UUID id);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND d.deletedAt IS NULL ORDER BY d.updatedAt DESC")
    List<Design> findAllByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND d.deletedAt IS NULL ORDER BY d.updatedAt DESC")
    Page<Design> findAllByProjectId(@Param("projectId") UUID projectId, Pageable pageable);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND d.status = :status AND d.deletedAt IS NULL")
    List<Design> findAllByProjectIdAndStatus(@Param("projectId") UUID projectId, @Param("status") String status);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND d.designType = :designType AND d.deletedAt IS NULL")
    List<Design> findAllByProjectIdAndType(@Param("projectId") UUID projectId, @Param("designType") Design.DesignType designType);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND d.folderId = :folderId AND d.deletedAt IS NULL")
    List<Design> findAllByProjectIdAndFolderId(@Param("projectId") UUID projectId, @Param("folderId") UUID folderId);

    @Query("SELECT d FROM Design d WHERE d.project.id = :projectId AND (d.name ILIKE %:keyword% OR d.description ILIKE %:keyword%) AND d.deletedAt IS NULL")
    List<Design> searchByKeyword(@Param("projectId") UUID projectId, @Param("keyword") String keyword);

    @Query("SELECT COUNT(d) FROM Design d WHERE d.project.id = :projectId AND d.deletedAt IS NULL")
    long countByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT d FROM Design d WHERE d.lockedBy = :userId AND d.lockExpiresAt > :now")
    List<Design> findLockedByUserId(@Param("userId") UUID userId, @Param("now") LocalDateTime now);

    @Query("SELECT d FROM Design d WHERE d.lockExpiresAt < :now AND d.lockedBy IS NOT NULL")
    List<Design> findExpiredLocks(@Param("now") LocalDateTime now);

    @Modifying
    @Query("UPDATE Design d SET d.lockedBy = :userId, d.lockedAt = :lockedAt, d.lockExpiresAt = :expiresAt WHERE d.id = :designId AND d.lockedBy IS NULL")
    int acquireLock(@Param("designId") UUID designId, @Param("userId") UUID userId, @Param("lockedAt") LocalDateTime lockedAt, @Param("expiresAt") LocalDateTime expiresAt);

    @Modifying
    @Query("UPDATE Design d SET d.lockedBy = NULL, d.lockedAt = NULL, d.lockExpiresAt = NULL WHERE d.id = :designId")
    void releaseLock(@Param("designId") UUID designId);

    @Modifying
    @Query("UPDATE Design d SET d.lockedBy = NULL, d.lockedAt = NULL, d.lockExpiresAt = NULL WHERE d.lockExpiresAt < :now")
    void releaseExpiredLocks(@Param("now") LocalDateTime now);

    @Modifying
    @Query("UPDATE Design d SET d.version = d.version + 1, d.updatedBy = :userId WHERE d.id = :designId")
    void incrementVersion(@Param("designId") UUID designId, @Param("userId") UUID userId);

    @Modifying
    @Query("UPDATE Design d SET d.deletedAt = :deletedAt WHERE d.id = :designId")
    void softDelete(@Param("designId") UUID designId, @Param("deletedAt") LocalDateTime deletedAt);

    @Modifying
    @Query("UPDATE Design d SET d.folderId = :folderId WHERE d.id = :designId")
    void moveToFolder(@Param("designId") UUID designId, @Param("folderId") UUID folderId);

    @Query("SELECT d FROM Design d WHERE d.createdBy = :userId AND d.deletedAt IS NULL ORDER BY d.createdAt DESC")
    List<Design> findAllByCreatedBy(@Param("userId") UUID userId);
}
