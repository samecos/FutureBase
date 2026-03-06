package com.archplatform.version.service;

import com.archplatform.version.dto.BranchDTO;
import com.archplatform.version.dto.CreateBranchRequest;
import com.archplatform.version.entity.Branch;
import com.archplatform.version.entity.Version;
import com.archplatform.version.exception.BranchNotFoundException;
import com.archplatform.version.exception.VersionNotFoundException;
import com.archplatform.version.repository.BranchRepository;
import com.archplatform.version.repository.VersionRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class BranchService {

    private final BranchRepository branchRepository;
    private final VersionRepository versionRepository;

    @Value("${version-control.max-branches-per-design:20}")
    private int maxBranchesPerDesign;

    @Transactional(readOnly = true)
    public BranchDTO getBranch(UUID branchId) {
        Branch branch = branchRepository.findByIdAndDeletedAtIsNull(branchId)
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + branchId));
        return mapToDTO(branch);
    }

    @Transactional(readOnly = true)
    public List<BranchDTO> getBranchesByDesign(UUID designId) {
        return branchRepository.findAllByDesignId(designId).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public BranchDTO getDefaultBranch(UUID designId) {
        Branch branch = branchRepository.findDefaultBranchByDesignId(designId)
            .orElseThrow(() -> new BranchNotFoundException("No default branch found for design: " + designId));
        return mapToDTO(branch);
    }

    @Transactional
    public BranchDTO createBranch(UUID tenantId, UUID userId, CreateBranchRequest request) {
        // Check branch limit
        long branchCount = branchRepository.countByDesignIdAndDeletedAtIsNull(request.getDesignId());
        if (branchCount >= maxBranchesPerDesign) {
            throw new IllegalStateException("Maximum number of branches reached for this design");
        }

        // Check for duplicate name
        if (branchRepository.existsByDesignIdAndNameAndDeletedAtIsNull(request.getDesignId(), request.getName())) {
            throw new IllegalArgumentException("Branch with name '" + request.getName() + "' already exists");
        }

        // Determine parent version
        UUID parentVersionId = request.getParentVersionId();
        if (parentVersionId == null && request.getParentBranchId() != null) {
            // Get head version of parent branch
            Branch parentBranch = branchRepository.findByIdAndDeletedAtIsNull(request.getParentBranchId())
                .orElseThrow(() -> new BranchNotFoundException("Parent branch not found"));
            parentVersionId = parentBranch.getHeadVersionId();
        }

        Integer baseVersionNumber = 0;
        if (parentVersionId != null) {
            Version parentVersion = versionRepository.findById(parentVersionId)
                .orElseThrow(() -> new VersionNotFoundException("Parent version not found"));
            baseVersionNumber = parentVersion.getVersionNumber();
        }

        Branch branch = Branch.builder()
            .tenantId(tenantId)
            .designId(request.getDesignId())
            .name(request.getName())
            .description(request.getDescription())
            .status(Branch.BranchStatus.ACTIVE)
            .parentBranchId(request.getParentBranchId())
            .parentVersionId(parentVersionId)
            .baseVersionNumber(baseVersionNumber)
            .versionCount(0)
            .isDefault(request.getIsDefault() != null ? request.getIsDefault() : false)
            .isProtected(request.getIsProtected() != null ? request.getIsProtected() : false)
            .createdBy(userId)
            .build();

        // If this is set as default, clear other defaults
        if (Boolean.TRUE.equals(branch.getIsDefault())) {
            branchRepository.clearDefaultFlag(request.getDesignId());
        }

        Branch saved = branchRepository.save(branch);
        log.info("Created branch: {} for design: {} by user: {}", saved.getId(), request.getDesignId(), userId);

        return mapToDTO(saved);
    }

    @Transactional
    public BranchDTO updateBranch(UUID branchId, UUID userId, String name, String description) {
        Branch branch = branchRepository.findByIdAndDeletedAtIsNull(branchId)
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + branchId));

        if (Boolean.TRUE.equals(branch.getIsProtected())) {
            throw new IllegalStateException("Cannot modify protected branch");
        }

        if (name != null && !name.equals(branch.getName())) {
            if (branchRepository.existsByDesignIdAndNameAndDeletedAtIsNull(branch.getDesignId(), name)) {
                throw new IllegalArgumentException("Branch with name '" + name + "' already exists");
            }
            branch.setName(name);
        }

        if (description != null) {
            branch.setDescription(description);
        }

        Branch saved = branchRepository.save(branch);
        return mapToDTO(saved);
    }

    @Transactional
    public void deleteBranch(UUID branchId, UUID userId) {
        Branch branch = branchRepository.findByIdAndDeletedAtIsNull(branchId)
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + branchId));

        if (Boolean.TRUE.equals(branch.getIsProtected())) {
            throw new IllegalStateException("Cannot delete protected branch");
        }

        if (Boolean.TRUE.equals(branch.getIsDefault())) {
            throw new IllegalStateException("Cannot delete default branch");
        }

        branchRepository.softDelete(branchId, LocalDateTime.now());
        log.info("Deleted branch: {} by user: {}", branchId, userId);
    }

    @Transactional
    public void setDefaultBranch(UUID designId, UUID branchId) {
        Branch branch = branchRepository.findByIdAndDeletedAtIsNull(branchId)
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + branchId));

        branchRepository.clearDefaultFlag(designId);
        branchRepository.setDefaultFlag(branchId);
        
        log.info("Set default branch: {} for design: {}", branchId, designId);
    }

    @Transactional(readOnly = true)
    public List<BranchDTO> getBranchHistory(UUID branchId) {
        // Get branch ancestry
        Branch current = branchRepository.findByIdAndDeletedAtIsNull(branchId)
            .orElseThrow(() -> new BranchNotFoundException("Branch not found: " + branchId));

        // This is a simplified implementation
        // In a real scenario, you'd traverse the parent chain
        return List.of(mapToDTO(current));
    }

    private BranchDTO mapToDTO(Branch branch) {
        String parentBranchName = null;
        if (branch.getParentBranchId() != null) {
            parentBranchName = branchRepository.findById(branch.getParentBranchId())
                .map(Branch::getName)
                .orElse(null);
        }

        Integer parentVersionNumber = null;
        if (branch.getParentVersionId() != null) {
            parentVersionNumber = versionRepository.findById(branch.getParentVersionId())
                .map(Version::getVersionNumber)
                .orElse(null);
        }

        Integer headVersionNumber = null;
        if (branch.getHeadVersionId() != null) {
            headVersionNumber = versionRepository.findById(branch.getHeadVersionId())
                .map(Version::getVersionNumber)
                .orElse(null);
        }

        return BranchDTO.builder()
            .id(branch.getId())
            .designId(branch.getDesignId())
            .name(branch.getName())
            .description(branch.getDescription())
            .status(branch.getStatus().name())
            .parentBranchId(branch.getParentBranchId())
            .parentBranchName(parentBranchName)
            .parentVersionId(branch.getParentVersionId())
            .parentVersionNumber(parentVersionNumber)
            .headVersionId(branch.getHeadVersionId())
            .headVersionNumber(headVersionNumber)
            .baseVersionNumber(branch.getBaseVersionNumber())
            .versionCount(branch.getVersionCount())
            .isDefault(branch.getIsDefault())
            .isProtected(branch.getIsProtected())
            .createdBy(branch.getCreatedBy())
            .createdAt(branch.getCreatedAt())
            .updatedAt(branch.getUpdatedAt())
            .build();
    }
}
