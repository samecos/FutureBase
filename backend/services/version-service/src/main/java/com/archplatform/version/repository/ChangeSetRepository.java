package com.archplatform.version.repository;

import com.archplatform.version.entity.ChangeSet;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.UUID;

@Repository
public interface ChangeSetRepository extends JpaRepository<ChangeSet, UUID> {

    @Query("SELECT cs FROM ChangeSet cs WHERE cs.versionId = :versionId ORDER BY cs.createdAt")
    List<ChangeSet> findAllByVersionId(@Param("versionId") UUID versionId);

    @Query("SELECT cs FROM ChangeSet cs WHERE cs.designId = :designId AND cs.entityId = :entityId ORDER BY cs.createdAt DESC")
    List<ChangeSet> findAllByDesignIdAndEntityId(@Param("designId") UUID designId, @Param("entityId") UUID entityId);

    @Query("SELECT cs FROM ChangeSet cs WHERE cs.designId = :designId AND cs.entityType = :entityType ORDER BY cs.createdAt DESC")
    List<ChangeSet> findAllByDesignIdAndEntityType(@Param("designId") UUID designId, @Param("entityType") String entityType);

    @Query("SELECT cs FROM ChangeSet cs WHERE cs.designId = :designId AND cs.changeType = :changeType ORDER BY cs.createdAt DESC")
    List<ChangeSet> findAllByDesignIdAndChangeType(@Param("designId") UUID designId, @Param("changeType") ChangeSet.ChangeType changeType);

    @Query("SELECT COUNT(cs) FROM ChangeSet cs WHERE cs.versionId = :versionId")
    long countByVersionId(@Param("versionId") UUID versionId);

    @Query("SELECT cs FROM ChangeSet cs WHERE cs.designId = :designId AND cs.createdAt BETWEEN :startDate AND :endDate ORDER BY cs.createdAt")
    List<ChangeSet> findAllByDesignIdAndDateRange(@Param("designId") UUID designId, 
                                                   @Param("startDate") LocalDateTime startDate, 
                                                   @Param("endDate") LocalDateTime endDate);
}
