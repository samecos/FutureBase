package com.archplatform.project.repository;

import com.archplatform.project.entity.ProjectFolder;
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
public interface ProjectFolderRepository extends JpaRepository<ProjectFolder, UUID> {

    @Query("SELECT f FROM ProjectFolder f WHERE f.id = :id AND f.deletedAt IS NULL")
    Optional<ProjectFolder> findActiveById(@Param("id") UUID id);

    @Query("SELECT f FROM ProjectFolder f WHERE f.project.id = :projectId AND f.deletedAt IS NULL ORDER BY f.name ASC")
    List<ProjectFolder> findAllByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT f FROM ProjectFolder f WHERE f.project.id = :projectId AND f.parentFolderId IS NULL AND f.deletedAt IS NULL ORDER BY f.name ASC")
    List<ProjectFolder> findRootFoldersByProjectId(@Param("projectId") UUID projectId);

    @Query("SELECT f FROM ProjectFolder f WHERE f.parentFolderId = :parentId AND f.deletedAt IS NULL ORDER BY f.name ASC")
    List<ProjectFolder> findAllByParentFolderId(@Param("parentId") UUID parentId);

    @Query("SELECT COUNT(f) FROM ProjectFolder f WHERE f.project.id = :projectId AND f.deletedAt IS NULL")
    long countByProjectId(@Param("projectId") UUID projectId);

    @Modifying
    @Query("UPDATE ProjectFolder f SET f.itemCount = f.itemCount + 1 WHERE f.id = :folderId")
    void incrementItemCount(@Param("folderId") UUID folderId);

    @Modifying
    @Query("UPDATE ProjectFolder f SET f.itemCount = f.itemCount - 1 WHERE f.id = :folderId AND f.itemCount > 0")
    void decrementItemCount(@Param("folderId") UUID folderId);

    @Modifying
    @Query("UPDATE ProjectFolder f SET f.deletedAt = :deletedAt WHERE f.id = :folderId")
    void softDelete(@Param("folderId") UUID folderId, @Param("deletedAt") LocalDateTime deletedAt);

    @Modifying
    @Query("UPDATE ProjectFolder f SET f.parentFolderId = :newParentId WHERE f.id = :folderId")
    void moveFolder(@Param("folderId") UUID folderId, @Param("newParentId") UUID newParentId);

    boolean existsByProjectIdAndNameAndParentFolderId(UUID projectId, String name, UUID parentFolderId);
}
