package com.archplatform.project.service;

import com.archplatform.project.dto.*;
import com.archplatform.project.entity.*;
import com.archplatform.project.event.ProjectEvent;
import com.archplatform.project.exception.*;
import com.archplatform.project.repository.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class DesignService {

    private final DesignRepository designRepository;
    private final ProjectRepository projectRepository;
    private final ProjectMemberRepository projectMemberRepository;
    private final ProjectFolderRepository folderRepository;
    private final KafkaTemplate<String, ProjectEvent> kafkaTemplate;

    @Transactional(readOnly = true)
    public DesignDTO getDesignById(UUID designId, UUID userId) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        validateDesignAccess(design, userId);

        return mapToDTO(design);
    }

    @Transactional(readOnly = true)
    public List<DesignDTO> getDesignsByProject(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectAccess(project, userId);

        return designRepository.findAllByProjectId(projectId).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public Page<DesignDTO> getDesignsByProject(UUID projectId, UUID userId, Pageable pageable) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectAccess(project, userId);

        return designRepository.findAllByProjectId(projectId, pageable)
            .map(this::mapToDTO);
    }

    @Transactional(readOnly = true)
    public List<DesignDTO> searchDesigns(UUID projectId, String keyword, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectAccess(project, userId);

        return designRepository.searchByKeyword(projectId, keyword).stream()
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public DesignDTO createDesign(UUID projectId, UUID userId, CreateDesignRequest request) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, userId, ProjectMember.MemberPermission.WRITE);

        Design design = Design.builder()
            .project(project)
            .tenantId(project.getTenantId())
            .name(request.getName())
            .description(request.getDescription())
            .designType(request.getDesignType() != null ? 
                Design.DesignType.valueOf(request.getDesignType().toUpperCase()) : null)
            .fileFormat(request.getFileFormat())
            .folderId(request.getFolderId())
            .createdBy(userId)
            .tags(request.getTags() != null ? String.join(",", request.getTags()) : null)
            .metadata(request.getMetadata())
            .build();

        Design savedDesign = designRepository.save(design);
        projectRepository.incrementDesignCount(projectId);

        if (request.getFolderId() != null) {
            folderRepository.incrementItemCount(request.getFolderId());
        }

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.DESIGN_CREATED,
            project.getId(),
            project.getTenantId(),
            userId,
            Map.of("designId", savedDesign.getId().toString(), "name", savedDesign.getName())
        ));

        log.info("Created design: {} in project: {} by user: {}", savedDesign.getId(), projectId, userId);
        return mapToDTO(savedDesign);
    }

    @Transactional
    public DesignDTO updateDesign(UUID designId, UUID userId, CreateDesignRequest request) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        validateProjectPermission(design.getProject(), userId, ProjectMember.MemberPermission.WRITE);

        // Check if design is locked by another user
        if (design.isLocked() && !design.getLockedBy().equals(userId)) {
            throw new UnauthorizedAccessException("Design is locked by another user");
        }

        if (request.getName() != null) {
            design.setName(request.getName());
        }
        if (request.getDescription() != null) {
            design.setDescription(request.getDescription());
        }
        if (request.getDesignType() != null) {
            design.setDesignType(Design.DesignType.valueOf(request.getDesignType().toUpperCase()));
        }
        design.setUpdatedBy(userId);

        Design updatedDesign = designRepository.save(design);

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.DESIGN_UPDATED,
            design.getProject().getId(),
            design.getTenantId(),
            userId,
            Map.of("designId", designId.toString(), "name", updatedDesign.getName())
        ));

        return mapToDTO(updatedDesign);
    }

    @Transactional
    public void deleteDesign(UUID designId, UUID userId) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        validateProjectPermission(design.getProject(), userId, ProjectMember.MemberPermission.DELETE);

        UUID folderId = design.getFolderId();
        UUID projectId = design.getProject().getId();

        designRepository.softDelete(designId, LocalDateTime.now());
        projectRepository.decrementDesignCount(projectId);

        if (folderId != null) {
            folderRepository.decrementItemCount(folderId);
        }

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.DESIGN_DELETED,
            projectId,
            design.getTenantId(),
            userId,
            Map.of("designId", designId.toString(), "name", design.getName())
        ));

        log.info("Deleted design: {} by user: {}", designId, userId);
    }

    @Transactional
    public boolean acquireLock(UUID designId, UUID userId, int durationMinutes) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        validateProjectPermission(design.getProject(), userId, ProjectMember.MemberPermission.WRITE);

        // Release any expired locks first
        designRepository.releaseExpiredLocks(LocalDateTime.now());

        // Try to acquire lock
        LocalDateTime lockedAt = LocalDateTime.now();
        LocalDateTime expiresAt = lockedAt.plusMinutes(durationMinutes);
        
        int updated = designRepository.acquireLock(designId, userId, lockedAt, expiresAt);
        
        if (updated > 0) {
            log.info("User {} acquired lock on design {} until {}", userId, designId, expiresAt);
            return true;
        }

        // Lock already held by someone else
        Design currentDesign = designRepository.findById(designId).orElseThrow();
        if (currentDesign.getLockedBy().equals(userId)) {
            // Extend existing lock
            currentDesign.lock(userId, durationMinutes);
            designRepository.save(currentDesign);
            return true;
        }

        return false;
    }

    @Transactional
    public void releaseLock(UUID designId, UUID userId) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        if (design.getLockedBy() != null && design.getLockedBy().equals(userId)) {
            designRepository.releaseLock(designId);
            log.info("User {} released lock on design {}", userId, designId);
        }
    }

    @Transactional
    public void moveDesign(UUID designId, UUID targetFolderId, UUID userId) {
        Design design = designRepository.findActiveById(designId)
            .orElseThrow(() -> new DesignNotFoundException("Design not found: " + designId));

        validateProjectPermission(design.getProject(), userId, ProjectMember.MemberPermission.WRITE);

        UUID oldFolderId = design.getFolderId();
        
        designRepository.moveToFolder(designId, targetFolderId);

        if (oldFolderId != null) {
            folderRepository.decrementItemCount(oldFolderId);
        }
        if (targetFolderId != null) {
            folderRepository.incrementItemCount(targetFolderId);
        }
    }

    // Helper methods
    private void validateProjectAccess(Project project, UUID userId) {
        if (project.getOwnerId().equals(userId)) {
            return;
        }
        if (!projectMemberRepository.existsByProjectIdAndUserId(project.getId(), userId)) {
            throw new UnauthorizedAccessException("You don't have access to this project");
        }
    }

    private void validateProjectPermission(Project project, UUID userId, ProjectMember.MemberPermission permission) {
        if (project.getOwnerId().equals(userId)) {
            return;
        }
        ProjectMember member = projectMemberRepository.findByProjectIdAndUserId(project.getId(), userId)
            .orElseThrow(() -> new UnauthorizedAccessException("You are not a member of this project"));
        
        if (!member.hasPermission(permission)) {
            throw new UnauthorizedAccessException("You don't have permission to perform this action");
        }
    }

    private void validateDesignAccess(Design design, UUID userId) {
        validateProjectAccess(design.getProject(), userId);
    }

    private void publishEvent(ProjectEvent event) {
        try {
            kafkaTemplate.send("project-events", event.getProjectId().toString(), event);
        } catch (Exception e) {
            log.warn("Failed to publish project event: {}", e.getMessage());
        }
    }

    private DesignDTO mapToDTO(Design design) {
        return DesignDTO.builder()
            .id(design.getId())
            .projectId(design.getProject().getId())
            .name(design.getName())
            .description(design.getDescription())
            .designType(design.getDesignType() != null ? design.getDesignType().name() : null)
            .fileFormat(design.getFileFormat())
            .version(design.getVersion())
            .status(design.getStatus())
            .thumbnailUrl(design.getThumbnailUrl())
            .fileSizeBytes(design.getFileSizeBytes())
            .createdBy(design.getCreatedBy())
            .updatedBy(design.getUpdatedBy())
            .lockedBy(design.getLockedBy())
            .lockedAt(design.getLockedAt())
            .lockExpiresAt(design.getLockExpiresAt())
            .isLocked(design.isLocked())
            .folderId(design.getFolderId())
            .tags(design.getTags() != null ? List.of(design.getTags().split(",")) : null)
            .createdAt(design.getCreatedAt())
            .updatedAt(design.getUpdatedAt())
            .build();
    }
}
