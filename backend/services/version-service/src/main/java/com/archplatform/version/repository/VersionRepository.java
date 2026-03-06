package com.archplatform.version.repository;

import com.archplatform.version.entity.Version;
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
public interface VersionRepository extends JpaRepository<Version, UUID> {

    Optional<Version> findById(UUID id);

    @Query("SELECT v FROM Version v WHERE v.branch.id = :branchId ORDER BY v.versionNumber DESC")
    List<Version> findAllByBranchId(@Param("branchId") UUID branchId);

    @Query("SELECT v FROM Version v WHERE v.branch.id = :branchId ORDER BY v.versionNumber DESC")
    Page<Version> findAllByBranchId(@Param("branchId") UUID branchId, Pageable pageable);

    @Query("SELECT v FROM Version v WHERE v.designId = :designId ORDER BY v.createdAt DESC")
    List<Version> findAllByDesignId(@Param("designId") UUID designId);

    @Query("SELECT v FROM Version v WHERE v.designId = :designId AND v.status = :status ORDER BY v.createdAt DESC")
    List<Version> findAllByDesignIdAndStatus(@Param("designId") UUID designId, @Param("status") Version.VersionStatus status);

    @Query("SELECT v FROM Version v WHERE v.branch.id = :branchId AND v.versionNumber = :versionNumber")
    Optional<Version> findByBranchIdAndVersionNumber(@Param("branchId") UUID branchId, @Param("versionNumber") Integer versionNumber);

    @Query("SELECT v FROM Version v WHERE v.branch.id = :branchId AND v.status = 'COMMITTED' ORDER BY v.versionNumber DESC")
    List<Version> findCommittedVersionsByBranchId(@Param("branchId") UUID branchId);

    @Query("SELECT MAX(v.versionNumber) FROM Version v WHERE v.branch.id = :branchId")
    Integer findMaxVersionNumberByBranchId(@Param("branchId") UUID branchId);

    @Query("SELECT v FROM Version v WHERE v.previousVersionId = :versionId")
    List<Version> findChildren(@Param("versionId") UUID versionId);

    @Modifying
    @Query("UPDATE Version v SET v.status = :status, v.committedAt = :committedAt, v.committedBy = :committedBy WHERE v.id = :versionId")
    void commitVersion(@Param("versionId") UUID versionId, @Param("status") Version.VersionStatus status, 
                       @Param("committedAt") LocalDateTime committedAt, @Param("committedBy") UUID committedBy);

    @Modifying
    @Query("UPDATE Version v SET v.status = 'ARCHIVED' WHERE v.id = :versionId")
    void archiveVersion(@Param("versionId") UUID versionId);

    long countByBranchId(UUID branchId);

    @Query("SELECT v FROM Version v WHERE v.isTagged = true AND v.tags LIKE %:tag%")
    List<Version> findByTag(@Param("tag") String tag);
}
