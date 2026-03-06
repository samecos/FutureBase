package com.archplatform.version.repository;

import com.archplatform.version.entity.MergeRequest;
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
public interface MergeRequestRepository extends JpaRepository<MergeRequest, UUID> {

    @Query("SELECT mr FROM MergeRequest mr WHERE mr.designId = :designId ORDER BY mr.createdAt DESC")
    List<MergeRequest> findAllByDesignId(@Param("designId") UUID designId);

    @Query("SELECT mr FROM MergeRequest mr WHERE mr.sourceBranchId = :branchId OR mr.targetBranchId = :branchId ORDER BY mr.createdAt DESC")
    List<MergeRequest> findAllByBranchId(@Param("branchId") UUID branchId);

    @Query("SELECT mr FROM MergeRequest mr WHERE mr.designId = :designId AND mr.status = :status ORDER BY mr.createdAt DESC")
    List<MergeRequest> findAllByDesignIdAndStatus(@Param("designId") UUID designId, @Param("status") MergeRequest.MergeStatus status);

    @Query("SELECT mr FROM MergeRequest mr WHERE mr.sourceBranchId = :sourceBranchId AND mr.targetBranchId = :targetBranchId AND mr.status IN ('OPEN', 'CONFLICTS')")
    Optional<MergeRequest> findOpenBySourceAndTarget(@Param("sourceBranchId") UUID sourceBranchId, @Param("targetBranchId") UUID targetBranchId);

    @Query("SELECT COUNT(mr) FROM MergeRequest mr WHERE mr.designId = :designId AND mr.status = :status")
    long countByDesignIdAndStatus(@Param("designId") UUID designId, @Param("status") MergeRequest.MergeStatus status);

    @Modifying
    @Query("UPDATE MergeRequest mr SET mr.status = :status, mr.mergedBy = :mergedBy, mr.mergedAt = :mergedAt, mr.resultVersionId = :resultVersionId WHERE mr.id = :mergeRequestId")
    void markAsMerged(@Param("mergeRequestId") UUID mergeRequestId, @Param("status") MergeRequest.MergeStatus status,
                      @Param("mergedBy") UUID mergedBy, @Param("mergedAt") LocalDateTime mergedAt, 
                      @Param("resultVersionId") UUID resultVersionId);

    @Modifying
    @Query("UPDATE MergeRequest mr SET mr.status = 'CLOSED', mr.closedAt = :closedAt WHERE mr.id = :mergeRequestId")
    void closeMergeRequest(@Param("mergeRequestId") UUID mergeRequestId, @Param("closedAt") LocalDateTime closedAt);

    @Modifying
    @Query("UPDATE MergeRequest mr SET mr.conflictCount = :conflictCount, mr.status = :status WHERE mr.id = :mergeRequestId")
    void updateConflictInfo(@Param("mergeRequestId") UUID mergeRequestId, @Param("conflictCount") Integer conflictCount,
                            @Param("status") MergeRequest.MergeStatus status);
}
