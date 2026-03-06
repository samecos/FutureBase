package com.archplatform.version.repository;

import com.archplatform.version.entity.Snapshot;
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
public interface SnapshotRepository extends JpaRepository<Snapshot, UUID> {

    Optional<Snapshot> findByVersionId(UUID versionId);

    @Query("SELECT s FROM Snapshot s WHERE s.designId = :designId ORDER BY s.createdAt DESC")
    List<Snapshot> findAllByDesignId(@Param("designId") UUID designId);

    @Query("SELECT s FROM Snapshot s WHERE s.tenantId = :tenantId AND s.retentionUntil < :now")
    List<Snapshot> findExpiredSnapshots(@Param("tenantId") UUID tenantId, @Param("now") LocalDateTime now);

    @Query("SELECT s FROM Snapshot s WHERE s.lastAccessedAt < :date AND s.accessCount < :minAccessCount")
    List<Snapshot> findColdSnapshots(@Param("date") LocalDateTime date, @Param("minAccessCount") Long minAccessCount);

    @Modifying
    @Query("UPDATE Snapshot s SET s.accessCount = s.accessCount + 1, s.lastAccessedAt = :accessedAt WHERE s.id = :snapshotId")
    void recordAccess(@Param("snapshotId") UUID snapshotId, @Param("accessedAt") LocalDateTime accessedAt);

    long countByDesignId(UUID designId);
}
