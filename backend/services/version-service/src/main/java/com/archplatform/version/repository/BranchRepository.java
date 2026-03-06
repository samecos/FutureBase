package com.archplatform.version.repository;

import com.archplatform.version.entity.Branch;
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
public interface BranchRepository extends JpaRepository<Branch, UUID> {

    Optional<Branch> findByIdAndDeletedAtIsNull(UUID id);

    @Query("SELECT b FROM Branch b WHERE b.designId = :designId AND b.deletedAt IS NULL ORDER BY b.isDefault DESC, b.createdAt DESC")
    List<Branch> findAllByDesignId(@Param("designId") UUID designId);

    @Query("SELECT b FROM Branch b WHERE b.designId = :designId AND b.status = :status AND b.deletedAt IS NULL")
    List<Branch> findAllByDesignIdAndStatus(@Param("designId") UUID designId, @Param("status") Branch.BranchStatus status);

    @Query("SELECT b FROM Branch b WHERE b.designId = :designId AND b.isDefault = true AND b.deletedAt IS NULL")
    Optional<Branch> findDefaultBranchByDesignId(@Param("designId") UUID designId);

    @Query("SELECT b FROM Branch b WHERE b.designId = :designId AND b.name = :name AND b.deletedAt IS NULL")
    Optional<Branch> findByDesignIdAndName(@Param("designId") UUID designId, @Param("name") String name);

    boolean existsByDesignIdAndNameAndDeletedAtIsNull(UUID designId, String name);

    long countByDesignIdAndDeletedAtIsNull(UUID designId);

    @Modifying
    @Query("UPDATE Branch b SET b.headVersionId = :versionId, b.versionCount = b.versionCount + 1 WHERE b.id = :branchId")
    void updateHeadVersion(@Param("branchId") UUID branchId, @Param("versionId") UUID versionId);

    @Modifying
    @Query("UPDATE Branch b SET b.status = :status WHERE b.id = :branchId")
    void updateStatus(@Param("branchId") UUID branchId, @Param("status") Branch.BranchStatus status);

    @Modifying
    @Query("UPDATE Branch b SET b.deletedAt = :deletedAt WHERE b.id = :branchId")
    void softDelete(@Param("branchId") UUID branchId, @Param("deletedAt") LocalDateTime deletedAt);

    @Modifying
    @Query("UPDATE Branch b SET b.isDefault = false WHERE b.designId = :designId")
    void clearDefaultFlag(@Param("designId") UUID designId);

    @Modifying
    @Query("UPDATE Branch b SET b.isDefault = true WHERE b.id = :branchId")
    void setDefaultFlag(@Param("branchId") UUID branchId);
}
