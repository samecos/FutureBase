package com.archplatform.version.service;

import com.archplatform.version.dto.CreateMergeRequest;
import com.archplatform.version.dto.MergeRequestDTO;
import com.archplatform.version.dto.VersionDiffDTO;
import com.archplatform.version.entity.Branch;
import com.archplatform.version.entity.MergeRequest;
import com.archplatform.version.entity.Version;
import com.archplatform.version.exception.BranchNotFoundException;
import com.archplatform.version.exception.MergeConflictException;
import com.archplatform.version.exception.VersionNotFoundException;
import com.archplatform.version.repository.BranchRepository;
import com.archplatform.version.repository.MergeRequestRepository;
import com.archplatform.version.repository.VersionRepository;
import com.archplatform.version.diff.VersionDiffService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class MergeService {

    private final MergeRequestRepository mergeRequestRepository;
    private final BranchRepository branchRepository;
    private final VersionRepository versionRepository;
    private final VersionDiffService diffService;
    private final VersionService versionService;

    @Transactional(readOnly = true)
    public MergeRequestDTO getMergeRequest(UUID mergeRequestId) {
        MergeRequest mr = mergeRequestRepository.findById(mergeRequestId)
            .orElseThrow(() -> new RuntimeException("Merge request not found: " + mergeRequestId));
        return mapToDTO(mr);
    }

    @Transactional(readOnly = true)
    public List<MergeRequestDTO> getMergeRequestsByDesign(UUID designId) {
        return mergeRequestRepository.findAllByDesignId(designId).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public List<MergeRequestDTO> getOpenMergeRequests(UUID designId) {
        return mergeRequestRepository.findAllByDesignIdAndStatus(designId, MergeRequest.MergeStatus.OPEN).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public MergeRequestDTO createMergeRequest(UUID tenantId, UUID userId, CreateMergeRequest request) {
        // Validate branches
        Branch sourceBranch = branchRepository.findByIdAndDeletedAtIsNull(request.getSourceBranchId())
            .orElseThrow(() -> new BranchNotFoundException("Source branch not found"));
        Branch targetBranch = branchRepository.findByIdAndDeletedAtIsNull(request.getTargetBranchId())
            .orElseThrow(() -> new BranchNotFoundException("Target branch not found"));

        if (sourceBranch.getDesignId().equals(targetBranch.getDesignId())) {
            throw new IllegalArgumentException("Source and target branches must belong to the same design");
        }

        // Check for existing open merge request
        mergeRequestRepository.findOpenBySourceAndTarget(request.getSourceBranchId(), request.getTargetBranchId())
            .ifPresent(mr -> {
                throw new IllegalStateException("An open merge request already exists for these branches");
            });

        // Get source version
        UUID sourceVersionId = sourceBranch.getHeadVersionId();
        if (sourceVersionId == null) {
            throw new IllegalStateException("Source branch has no versions");
        }

        // Detect conflicts
        int conflictCount = diffService.countConflicts(sourceVersionId, targetBranch.getHeadVersionId());
        MergeRequest.MergeStatus status = conflictCount > 0 ? MergeRequest.MergeStatus.CONFLICTS : MergeRequest.MergeStatus.OPEN;

        MergeRequest mr = MergeRequest.builder()
            .tenantId(tenantId)
            .designId(sourceBranch.getDesignId())
            .sourceBranchId(request.getSourceBranchId())
            .sourceVersionId(sourceVersionId)
            .targetBranchId(request.getTargetBranchId())
            .targetVersionId(targetBranch.getHeadVersionId())
            .title(request.getTitle())
            .description(request.getDescription())
            .status(status)
            .conflictCount(conflictCount)
            .createdBy(userId)
            .assignedTo(request.getAssignedTo())
            .build();

        MergeRequest saved = mergeRequestRepository.save(mr);
        log.info("Created merge request: {} from branch {} to branch {}", 
            saved.getId(), sourceBranch.getName(), targetBranch.getName());

        return mapToDTO(saved);
    }

    @Transactional
    public MergeRequestDTO performMerge(UUID mergeRequestId, UUID userId) {
        MergeRequest mr = mergeRequestRepository.findById(mergeRequestId)
            .orElseThrow(() -> new RuntimeException("Merge request not found: " + mergeRequestId));

        if (!mr.isOpen()) {
            throw new IllegalStateException("Merge request is not open");
        }

        if (mr.getConflictCount() > 0) {
            throw new MergeConflictException("Cannot merge: there are unresolved conflicts");
        }

        Branch targetBranch = branchRepository.findByIdAndDeletedAtIsNull(mr.getTargetBranchId())
            .orElseThrow(() -> new BranchNotFoundException("Target branch not found"));

        Version sourceVersion = versionRepository.findById(mr.getSourceVersionId())
            .orElseThrow(() -> new VersionNotFoundException("Source version not found"));

        // Create a new version in target branch with merged content
        // In a real implementation, you'd apply changes from source to target
        Version mergedVersion = Version.builder()
            .tenantId(mr.getTenantId())
            .designId(mr.getDesignId())
            .branch(targetBranch)
            .versionNumber((versionRepository.findMaxVersionNumberByBranchId(targetBranch.getId()) != null ? 
                versionRepository.findMaxVersionNumberByBranchId(targetBranch.getId()) : 0) + 1)
            .description("Merge from " + sourceVersion.getBranch().getName() + "/v" + sourceVersion.getVersionNumber())
            .status(Version.VersionStatus.COMMITTED)
            .snapshotId(sourceVersion.getSnapshotId())
            .previousVersionId(mr.getTargetVersionId())
            .parentVersionIds(mr.getSourceVersionId().toString())
            .committedBy(userId)
            .committedAt(LocalDateTime.now())
            .createdBy(userId)
            .build();

        Version savedVersion = versionRepository.save(mergedVersion);

        // Update merge request
        mergeRequestRepository.markAsMerged(mergeRequestId, MergeRequest.MergeStatus.MERGED, 
            userId, LocalDateTime.now(), savedVersion.getId());

        // Update branch head
        branchRepository.updateHeadVersion(targetBranch.getId(), savedVersion.getId());

        // Update source branch status
        branchRepository.updateStatus(mr.getSourceBranchId(), Branch.BranchStatus.MERGED);

        log.info("Merged request: {} into branch: {} by user: {}", mergeRequestId, targetBranch.getName(), userId);

        return mapToDTO(mergeRequestRepository.findById(mergeRequestId).orElseThrow());
    }

    @Transactional
    public void closeMergeRequest(UUID mergeRequestId, UUID userId) {
        MergeRequest mr = mergeRequestRepository.findById(mergeRequestId)
            .orElseThrow(() -> new RuntimeException("Merge request not found: " + mergeRequestId));

        if (mr.getStatus() == MergeRequest.MergeStatus.MERGED) {
            throw new IllegalStateException("Cannot close merged request");
        }

        mergeRequestRepository.closeMergeRequest(mergeRequestId, LocalDateTime.now());
        log.info("Closed merge request: {} by user: {}", mergeRequestId, userId);
    }

    @Transactional(readOnly = true)
    public VersionDiffDTO previewMerge(UUID sourceBranchId, UUID targetBranchId) {
        Branch sourceBranch = branchRepository.findByIdAndDeletedAtIsNull(sourceBranchId)
            .orElseThrow(() -> new BranchNotFoundException("Source branch not found"));
        Branch targetBranch = branchRepository.findByIdAndDeletedAtIsNull(targetBranchId)
            .orElseThrow(() -> new BranchNotFoundException("Target branch not found"));

        return diffService.compareVersions(
            sourceBranch.getHeadVersionId(),
            targetBranch.getHeadVersionId()
        );
    }

    @Transactional
    public MergeRequestDTO resolveConflicts(UUID mergeRequestId, String resolution, UUID userId) {
        MergeRequest mr = mergeRequestRepository.findById(mergeRequestId)
            .orElseThrow(() -> new RuntimeException("Merge request not found: " + mergeRequestId));

        mr.setConflictResolution(resolution);
        mr.setConflictCount(0);
        mr.setStatus(MergeRequest.MergeStatus.OPEN);

        MergeRequest saved = mergeRequestRepository.save(mr);
        log.info("Resolved conflicts for merge request: {} by user: {}", mergeRequestId, userId);

        return mapToDTO(saved);
    }

    private MergeRequestDTO mapToDTO(MergeRequest mr) {
        String sourceBranchName = branchRepository.findById(mr.getSourceBranchId())
            .map(Branch::getName).orElse("Unknown");
        String targetBranchName = branchRepository.findById(mr.getTargetBranchId())
            .map(Branch::getName).orElse("Unknown");

        Integer sourceVersionNumber = versionRepository.findById(mr.getSourceVersionId())
            .map(Version::getVersionNumber).orElse(null);
        Integer targetVersionNumber = mr.getTargetVersionId() != null ? 
            versionRepository.findById(mr.getTargetVersionId()).map(Version::getVersionNumber).orElse(null) : null;

        return MergeRequestDTO.builder()
            .id(mr.getId())
            .designId(mr.getDesignId())
            .sourceBranchId(mr.getSourceBranchId())
            .sourceBranchName(sourceBranchName)
            .sourceVersionId(mr.getSourceVersionId())
            .sourceVersionNumber(sourceVersionNumber)
            .targetBranchId(mr.getTargetBranchId())
            .targetBranchName(targetBranchName)
            .targetVersionId(mr.getTargetVersionId())
            .targetVersionNumber(targetVersionNumber)
            .title(mr.getTitle())
            .description(mr.getDescription())
            .status(mr.getStatus().name())
            .conflictCount(mr.getConflictCount())
            .conflictResolution(mr.getConflictResolution() != null ? 
                new com.fasterxml.jackson.databind.ObjectMapper().readTree(mr.getConflictResolution()) : null)
            .createdBy(mr.getCreatedBy())
            .assignedTo(mr.getAssignedTo())
            .mergedBy(mr.getMergedBy())
            .mergedAt(mr.getMergedAt())
            .resultVersionId(mr.getResultVersionId())
            .createdAt(mr.getCreatedAt())
            .updatedAt(mr.getUpdatedAt())
            .build();
    }
}
