package com.archplatform.version.service;

import com.archplatform.version.dto.*;
import com.archplatform.version.entity.Branch;
import com.archplatform.version.entity.Version;
import com.archplatform.version.entity.ChangeSet;
import com.archplatform.version.exception.BranchNotFoundException;
import com.archplatform.version.exception.VersionNotFoundException;
import com.archplatform.version.repository.BranchRepository;
import com.archplatform.version.repository.VersionRepository;
import com.archplatform.version.repository.ChangeSetRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.Arrays;
import java.util.List;
import java.util.UUID;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class VersionService {

    private final VersionRepository versionRepository;
    private final BranchRepository branchRepository;
    private final ChangeSetRepository changeSetRepository;

    @Transactional(readOnly = true)
    public VersionDTO getVersion(UUID versionId) {
        Version version = versionRepository.findById(versionId)
            .orElseThrow(() -> new VersionNotFoundException("Version not found: " + versionId));
        return mapToDTO(version);
    }

    @Transactional(readOnly = true)
    public List<VersionDTO> getVersionsByBranch(UUID branchId) {
        return versionRepository.findAllByBranchId(branchId).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public Page<VersionDTO> getVersionsByBranch(UUID branchId, Pageable pageable) {
        return versionRepository.findAllByBranchId(branchId, pageable)
            .map(this::mapToDTO);
    }

    @Transactional(readOnly = true)
    public List<VersionDTO> getVersionsByDesign(UUID designId) {
        return versionRepository.findAllByDesignId(designId).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public VersionDTO getLatestVersion(UUID branchId) {
        List<Version> versions = versionRepository.findAllByBranchId(branchId);
        if (versions.isEmpty()) {
            throw new VersionNotFoundException("No versions found for branch: " + branchId);
        }
        return mapToDTO(versions.get(0));
    }

    @Transactional
    public VersionDTO createVersion(UUID tenantId, UUID userId, CreateVersionRequest request) {
        Branch branch = branchRepository.findByIdAndDeletedAtIsNull(request.getBranchId())
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + request.getBranchId()));

        // Get next version number
        Integer maxVersion = versionRepository.findMaxVersionNumberByBranchId(branch.getId());
        int nextVersionNumber = (maxVersion != null ? maxVersion : branch.getBaseVersionNumber()) + 1;

        // Get previous version
        UUID previousVersionId = branch.getHeadVersionId();

        Version version = Version.builder()
            .tenantId(tenantId)
            .designId(branch.getDesignId())
            .branch(branch)
            .versionNumber(nextVersionNumber)
            .versionName(request.getVersionName())
            .description(request.getDescription())
            .status(Version.VersionStatus.DRAFT)
            .snapshotId(request.getSnapshotId())
            .snapshotUrl(request.getSnapshotUrl())
            .snapshotSizeBytes(request.getSnapshotSizeBytes())
            .checksum(request.getChecksum())
            .previousVersionId(previousVersionId)
            .changeSummary(request.getChangeSummary())
            .changeCount(request.getChangeCount() != null ? request.getChangeCount() : 0)
            .createdBy(userId)
            .isTagged(request.getTags() != null && !request.getTags().isEmpty())
            .tags(request.getTags() != null ? String.join(",", request.getTags()) : null)
            .build();

        Version saved = versionRepository.save(version);

        // Update branch head
        branchRepository.updateHeadVersion(branch.getId(), saved.getId());

        log.info("Created version: {} for branch: {} by user: {}", saved.getId(), branch.getId(), userId);
        return mapToDTO(saved);
    }

    @Transactional
    public VersionDTO commitVersion(UUID versionId, UUID userId, String description) {
        Version version = versionRepository.findById(versionId)
            .orElseThrow(() -> new VersionNotFoundException("Version not found: " + versionId));

        if (version.isCommitted()) {
            throw new IllegalStateException("Version is already committed");
        }

        version.setDescription(description != null ? description : version.getDescription());
        versionRepository.commitVersion(versionId, Version.VersionStatus.COMMITTED, LocalDateTime.now(), userId);

        // Refresh entity
        Version committed = versionRepository.findById(versionId).orElseThrow();
        log.info("Committed version: {} by user: {}", versionId, userId);

        return mapToDTO(committed);
    }

    @Transactional
    public void deleteVersion(UUID versionId, UUID userId) {
        Version version = versionRepository.findById(versionId)
            .orElseThrow(() -> new VersionNotFoundException("Version not found: " + versionId));

        if (version.isCommitted()) {
            throw new IllegalStateException("Cannot delete committed version");
        }

        versionRepository.archiveVersion(versionId);
        log.info("Archived version: {} by user: {}", versionId, userId);
    }

    @Transactional(readOnly = true)
    public List<ChangeSetDTO> getVersionChanges(UUID versionId) {
        return changeSetRepository.findAllByVersionId(versionId).stream()
            .map(this::mapToChangeSetDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public List<VersionDTO> getVersionHistory(UUID designId, UUID startVersionId, Integer limit) {
        Version startVersion = versionRepository.findById(startVersionId)
            .orElseThrow(() -> new VersionNotFoundException("Version not found: " + startVersionId));

        // This is a simplified implementation
        // In a real scenario, you'd traverse the version chain
        return versionRepository.findAllByDesignId(designId).stream()
            .filter(v -> v.getCreatedAt().isBefore(startVersion.getCreatedAt()) || v.getId().equals(startVersionId))
            .limit(limit != null ? limit : 50)
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public VersionDTO rollbackToVersion(UUID versionId, UUID userId) {
        Version targetVersion = versionRepository.findById(versionId)
            .orElseThrow(() -> new VersionNotFoundException("Version not found: " + versionId));

        Branch branch = targetVersion.getBranch();

        // Create a new version that reverts to the target
        CreateVersionRequest request = CreateVersionRequest.builder()
            .branchId(branch.getId())
            .versionName("Rollback to v" + targetVersion.getVersionNumber())
            .description("Rollback to version " + targetVersion.getFullVersionName())
            .snapshotId(targetVersion.getSnapshotId())
            .build();

        return createVersion(targetVersion.getTenantId(), userId, request);
    }

    private VersionDTO mapToDTO(Version version) {
        Branch branch = version.getBranch();
        
        Integer previousVersionNumber = null;
        if (version.getPreviousVersionId() != null) {
            previousVersionNumber = versionRepository.findById(version.getPreviousVersionId())
                .map(Version::getVersionNumber)
                .orElse(null);
        }

        List<String> parentIds = null;
        if (version.getParentVersionIds() != null) {
            parentIds = Arrays.asList(version.getParentVersionIds().split(","));
        }

        List<String> tags = null;
        if (version.getTags() != null) {
            tags = Arrays.asList(version.getTags().split(","));
        }

        return VersionDTO.builder()
            .id(version.getId())
            .designId(version.getDesignId())
            .branchId(branch.getId())
            .branchName(branch.getName())
            .versionNumber(version.getVersionNumber())
            .fullVersionName(version.getFullVersionName())
            .versionName(version.getVersionName())
            .description(version.getDescription())
            .status(version.getStatus().name())
            .snapshotId(version.getSnapshotId())
            .snapshotUrl(version.getSnapshotUrl())
            .snapshotSizeBytes(version.getSnapshotSizeBytes())
            .checksum(version.getChecksum())
            .previousVersionId(version.getPreviousVersionId())
            .previousVersionNumber(previousVersionNumber)
            .parentVersionIds(parentIds)
            .changeSummary(version.getChangeSummary())
            .changeCount(version.getChangeCount())
            .createdBy(version.getCreatedBy())
            .committedBy(version.getCommittedBy())
            .committedAt(version.getCommittedAt())
            .createdAt(version.getCreatedAt())
            .isTagged(version.getIsTagged())
            .tags(tags)
            .build();
    }

    private ChangeSetDTO mapToChangeSetDTO(ChangeSet changeSet) {
        return ChangeSetDTO.builder()
            .id(changeSet.getId())
            .versionId(changeSet.getVersionId())
            .changeType(changeSet.getChangeType().name())
            .entityType(changeSet.getEntityType())
            .entityId(changeSet.getEntityId())
            .propertyName(changeSet.getPropertyName())
            .oldValue(changeSet.getOldValue())
            .newValue(changeSet.getNewValue())
            .description(changeSet.getDescription())
            .createdBy(changeSet.getCreatedBy())
            .createdAt(changeSet.getCreatedAt())
            .build();
    }
}
