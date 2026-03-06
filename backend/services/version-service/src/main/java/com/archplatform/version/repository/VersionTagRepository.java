package com.archplatform.version.repository;

import com.archplatform.version.entity.VersionTag;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface VersionTagRepository extends JpaRepository<VersionTag, UUID> {

    Optional<VersionTag> findByDesignIdAndTagName(UUID designId, String tagName);

    @Query("SELECT vt FROM VersionTag vt WHERE vt.designId = :designId ORDER BY vt.createdAt DESC")
    List<VersionTag> findAllByDesignId(@Param("designId") UUID designId);

    @Query("SELECT vt FROM VersionTag vt WHERE vt.versionId = :versionId")
    List<VersionTag> findAllByVersionId(@Param("versionId") UUID versionId);

    boolean existsByDesignIdAndTagName(UUID designId, String tagName);

    @Query("SELECT COUNT(vt) FROM VersionTag vt WHERE vt.designId = :designId")
    long countByDesignId(@Param("designId") UUID designId);
}
