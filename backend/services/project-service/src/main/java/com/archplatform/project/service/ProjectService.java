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
import org.springframework.util.StringUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class ProjectService {

    private final ProjectRepository projectRepository;
    private final ProjectMemberRepository projectMemberRepository;
    private final DesignRepository designRepository;
    private final KafkaTemplate<String, ProjectEvent> kafkaTemplate;

    @Transactional(readOnly = true)
    public ProjectDTO getProjectById(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));
        
        validateProjectAccess(project, userId);
        
        return mapToDTO(project);
    }

    @Transactional(readOnly = true)
    public List<ProjectDTO> getProjectsByTenant(UUID tenantId, UUID userId) {
        return projectRepository.findAllByTenantId(tenantId).stream()
            .filter(p -> hasProjectAccess(p, userId))
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public Page<ProjectDTO> getProjectsByTenant(UUID tenantId, UUID userId, Pageable pageable) {
        return projectRepository.findAllByTenantId(tenantId, pageable)
            .map(this::mapToDTO);
    }

    @Transactional(readOnly = true)
    public List<ProjectDTO> getProjectsByUser(UUID userId) {
        // Get projects where user is a member
        List<ProjectMember> memberships = projectMemberRepository.findAllByUserId(userId);
        return memberships.stream()
            .map(ProjectMember::getProject)
            .filter(p -> p.getDeletedAt() == null)
            .map(this::mapToDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public ProjectDTO createProject(UUID tenantId, UUID ownerId, CreateProjectRequest request) {
        Project project = Project.builder()
            .tenantId(tenantId)
            .name(request.getName())
            .description(request.getDescription())
            .ownerId(ownerId)
            .status(Project.ProjectStatus.ACTIVE)
            .visibility(request.getVisibility() != null ? 
                Project.ProjectVisibility.valueOf(request.getVisibility().toUpperCase()) : 
                Project.ProjectVisibility.PRIVATE)
            .startDate(request.getStartDate())
            .targetEndDate(request.getTargetEndDate())
            .tags(request.getTags() != null ? String.join(",", request.getTags()) : null)
            .build();

        Project savedProject = projectRepository.save(project);

        // Add owner as project member with OWNER role
        ProjectMember ownerMember = ProjectMember.builder()
            .project(savedProject)
            .userId(ownerId)
            .role(ProjectMember.MemberRole.OWNER)
            .joinedAt(LocalDateTime.now())
            .build();
        projectMemberRepository.save(ownerMember);

        // Publish event
        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.PROJECT_CREATED,
            savedProject.getId(),
            tenantId,
            ownerId,
            Map.of("name", savedProject.getName())
        ));

        log.info("Created project: {} by user: {}", savedProject.getId(), ownerId);
        return mapToDTO(savedProject);
    }

    @Transactional
    public ProjectDTO updateProject(UUID projectId, UUID userId, UpdateProjectRequest request) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, userId, ProjectMember.MemberPermission.WRITE);

        if (StringUtils.hasText(request.getName())) {
            project.setName(request.getName());
        }
        if (request.getDescription() != null) {
            project.setDescription(request.getDescription());
        }
        if (StringUtils.hasText(request.getVisibility())) {
            project.setVisibility(Project.ProjectVisibility.valueOf(request.getVisibility().toUpperCase()));
        }
        if (StringUtils.hasText(request.getStatus())) {
            Project.ProjectStatus newStatus = Project.ProjectStatus.valueOf(request.getStatus().toUpperCase());
            project.setStatus(newStatus);
            if (newStatus == Project.ProjectStatus.COMPLETED) {
                project.setCompletedAt(LocalDateTime.now());
            }
        }
        if (request.getStartDate() != null) {
            project.setStartDate(request.getStartDate());
        }
        if (request.getTargetEndDate() != null) {
            project.setTargetEndDate(request.getTargetEndDate());
        }
        if (request.getTags() != null) {
            project.setTags(String.join(",", request.getTags()));
        }

        Project updatedProject = projectRepository.save(project);

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.PROJECT_UPDATED,
            updatedProject.getId(),
            updatedProject.getTenantId(),
            userId,
            Map.of("name", updatedProject.getName())
        ));

        return mapToDTO(updatedProject);
    }

    @Transactional
    public void deleteProject(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, userId, ProjectMember.MemberPermission.DELETE_PROJECT);

        projectRepository.softDelete(projectId, LocalDateTime.now());

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.PROJECT_DELETED,
            project.getId(),
            project.getTenantId(),
            userId,
            Map.of("name", project.getName())
        ));

        log.info("Deleted project: {} by user: {}", projectId, userId);
    }

    @Transactional
    public void archiveProject(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, userId, ProjectMember.MemberPermission.WRITE);

        projectRepository.archiveProject(projectId, Project.ProjectStatus.ARCHIVED, LocalDateTime.now());

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.PROJECT_ARCHIVED,
            project.getId(),
            project.getTenantId(),
            userId,
            Map.of("name", project.getName())
        ));

        log.info("Archived project: {} by user: {}", projectId, userId);
    }

    @Transactional(readOnly = true)
    public List<ProjectMemberDTO> getProjectMembers(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectAccess(project, userId);

        return projectMemberRepository.findAllByProjectId(projectId).stream()
            .map(this::mapMemberToDTO)
            .collect(Collectors.toList());
    }

    @Transactional
    public void addProjectMember(UUID projectId, UUID invitedBy, AddProjectMemberRequest request) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, invitedBy, ProjectMember.MemberPermission.MANAGE_MEMBERS);

        if (projectMemberRepository.existsByProjectIdAndUserId(projectId, request.getUserId())) {
            throw new MemberAlreadyExistsException("User is already a member of this project");
        }

        ProjectMember.MemberRole role = request.getRole() != null ? 
            ProjectMember.MemberRole.valueOf(request.getRole().toUpperCase()) : 
            ProjectMember.MemberRole.VIEWER;

        ProjectMember member = ProjectMember.builder()
            .project(project)
            .userId(request.getUserId())
            .role(role)
            .invitedBy(invitedBy)
            .invitedAt(LocalDateTime.now())
            .joinedAt(LocalDateTime.now())
            .build();

        projectMemberRepository.save(member);
        projectRepository.incrementMemberCount(projectId);

        publishEvent(new ProjectEvent(
            ProjectEvent.EventType.MEMBER_ADDED,
            project.getId(),
            project.getTenantId(),
            invitedBy,
            Map.of("addedUserId", request.getUserId().toString(), "role", role.name())
        ));

        log.info("Added member {} to project {} by {}", request.getUserId(), projectId, invitedBy);
    }

    @Transactional
    public void removeProjectMember(UUID projectId, UUID memberId, UUID removedBy) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, removedBy, ProjectMember.MemberPermission.MANAGE_MEMBERS);

        ProjectMember member = projectMemberRepository.findById(memberId)
            .orElseThrow(() -> new ProjectNotFoundException("Member not found"));

        // Cannot remove owner
        if (member.getRole() == ProjectMember.MemberRole.OWNER) {
            throw new UnauthorizedAccessException("Cannot remove project owner");
        }

        projectMemberRepository.delete(member);
        projectRepository.decrementMemberCount(projectId);

        log.info("Removed member {} from project {} by {}", memberId, projectId, removedBy);
    }

    @Transactional
    public void updateMemberRole(UUID projectId, UUID memberId, String newRole, UUID updatedBy) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectPermission(project, updatedBy, ProjectMember.MemberPermission.MANAGE_MEMBERS);

        ProjectMember member = projectMemberRepository.findById(memberId)
            .orElseThrow(() -> new ProjectNotFoundException("Member not found"));

        // Cannot change owner's role
        if (member.getRole() == ProjectMember.MemberRole.OWNER) {
            throw new UnauthorizedAccessException("Cannot change project owner's role");
        }

        ProjectMember.MemberRole role = ProjectMember.MemberRole.valueOf(newRole.toUpperCase());
        projectMemberRepository.updateRole(memberId, role);

        log.info("Updated member {} role to {} in project {} by {}", memberId, newRole, projectId, updatedBy);
    }

    @Transactional(readOnly = true)
    public ProjectStatsDTO getProjectStats(UUID projectId, UUID userId) {
        Project project = projectRepository.findActiveById(projectId)
            .orElseThrow(() -> new ProjectNotFoundException("Project not found: " + projectId));

        validateProjectAccess(project, userId);

        List<Design> designs = designRepository.findAllByProjectId(projectId);
        
        Map<String, Integer> designsByType = designs.stream()
            .filter(d -> d.getDesignType() != null)
            .collect(Collectors.groupingBy(
                d -> d.getDesignType().name(),
                Collectors.counting()
            ))
            .entrySet().stream()
            .collect(Collectors.toMap(
                Map.Entry::getKey,
                e -> e.getValue().intValue()
            ));

        Map<String, Integer> designsByStatus = designs.stream()
            .collect(Collectors.groupingBy(
                Design::getStatus,
                Collectors.counting()
            ))
            .entrySet().stream()
            .collect(Collectors.toMap(
                Map.Entry::getKey,
                e -> e.getValue().intValue()
            ));

        return ProjectStatsDTO.builder()
            .projectId(project.getId())
            .projectName(project.getName())
            .totalDesigns(designs.size())
            .totalMembers(project.getMemberCount())
            .totalStorageBytes(project.getTotalStorageBytes())
            .designsByType(designsByType)
            .designsByStatus(designsByStatus)
            .lastActivityAt(project.getUpdatedAt())
            .activityScore(calculateActivityScore(designs))
            .build();
    }

    // Helper methods
    private void validateProjectAccess(Project project, UUID userId) {
        if (project.getVisibility() == Project.ProjectVisibility.PUBLIC) {
            return;
        }
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

    private boolean hasProjectAccess(Project project, UUID userId) {
        if (project.getVisibility() == Project.ProjectVisibility.PUBLIC) {
            return true;
        }
        if (project.getOwnerId().equals(userId)) {
            return true;
        }
        return projectMemberRepository.existsByProjectIdAndUserId(project.getId(), userId);
    }

    private void publishEvent(ProjectEvent event) {
        try {
            kafkaTemplate.send("project-events", event.getProjectId().toString(), event);
        } catch (Exception e) {
            log.warn("Failed to publish project event: {}", e.getMessage());
        }
    }

    private int calculateActivityScore(List<Design> designs) {
        if (designs.isEmpty()) {
            return 0;
        }
        // Simple scoring based on recent updates
        LocalDateTime oneWeekAgo = LocalDateTime.now().minusWeeks(1);
        long recentUpdates = designs.stream()
            .filter(d -> d.getUpdatedAt().isAfter(oneWeekAgo))
            .count();
        return Math.min(100, (int) (recentUpdates * 10));
    }

    private ProjectDTO mapToDTO(Project project) {
        return ProjectDTO.builder()
            .id(project.getId())
            .tenantId(project.getTenantId())
            .name(project.getName())
            .description(project.getDescription())
            .slug(project.getSlug())
            .status(project.getStatus().name())
            .visibility(project.getVisibility().name())
            .ownerId(project.getOwnerId())
            .thumbnailUrl(project.getThumbnailUrl())
            .tags(project.getTags() != null ? Arrays.asList(project.getTags().split(",")) : null)
            .startDate(project.getStartDate())
            .targetEndDate(project.getTargetEndDate())
            .designCount(project.getDesignCount())
            .memberCount(project.getMemberCount())
            .totalStorageBytes(project.getTotalStorageBytes())
            .createdAt(project.getCreatedAt())
            .updatedAt(project.getUpdatedAt())
            .build();
    }

    private ProjectMemberDTO mapMemberToDTO(ProjectMember member) {
        return ProjectMemberDTO.builder()
            .id(member.getId())
            .userId(member.getUserId())
            .role(member.getRole().name())
            .invitedBy(member.getInvitedBy())
            .joinedAt(member.getJoinedAt())
            .lastAccessedAt(member.getLastAccessedAt())
            .build();
    }
}
