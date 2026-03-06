package com.archplatform.project.integration;

import com.archplatform.project.entity.Project;
import com.archplatform.project.entity.ProjectMember;
import com.archplatform.project.entity.enums.ProjectRole;
import com.archplatform.project.entity.enums.ProjectStatus;
import com.archplatform.project.repository.ProjectMemberRepository;
import com.archplatform.project.repository.ProjectRepository;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.time.LocalDateTime;
import java.util.HashSet;
import java.util.Set;
import java.util.UUID;

/**
 * Helper class for creating test data in Project Service integration tests.
 */
@Component
public class TestDataHelper {

    @Autowired
    private ProjectRepository projectRepository;

    @Autowired
    private ProjectMemberRepository memberRepository;

    private final Set<UUID> createdProjectIds = new HashSet<>();
    private final Set<UUID> createdMemberIds = new HashSet<>();

    /**
     * Create a test project.
     */
    public Project createTestProject() {
        return createTestProject(UUID.randomUUID());
    }

    /**
     * Create a test project with specified owner.
     */
    public Project createTestProject(UUID ownerId) {
        return createTestProject("Test Project", "Test Description", ownerId);
    }

    /**
     * Create a test project with all specified values.
     */
    public Project createTestProject(String name, String description, UUID ownerId) {
        Project project = Project.builder()
                .id(UUID.randomUUID())
                .name(name)
                .description(description)
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();

        Project saved = projectRepository.save(project);
        createdProjectIds.add(saved.getId());

        // Add owner as member
        addMember(saved.getId(), ownerId, ProjectRole.OWNER);

        return saved;
    }

    /**
     * Create an archived project.
     */
    public Project createArchivedProject(UUID ownerId) {
        Project project = Project.builder()
                .id(UUID.randomUUID())
                .name("Archived Project")
                .description("This project is archived")
                .ownerId(ownerId)
                .status(ProjectStatus.ARCHIVED)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();

        Project saved = projectRepository.save(project);
        createdProjectIds.add(saved.getId());
        return saved;
    }

    /**
     * Add a member to a project.
     */
    public ProjectMember addMember(UUID projectId, UUID userId, ProjectRole role) {
        ProjectMember member = ProjectMember.builder()
                .id(UUID.randomUUID())
                .projectId(projectId)
                .userId(userId)
                .role(role)
                .joinedAt(LocalDateTime.now())
                .build();

        ProjectMember saved = memberRepository.save(member);
        createdMemberIds.add(saved.getId());
        return saved;
    }

    /**
     * Get project by ID.
     */
    public Project getProject(UUID projectId) {
        return projectRepository.findById(projectId).orElse(null);
    }

    /**
     * Clear all test data.
     */
    public void clearAll() {
        // Delete in correct order to avoid FK constraints
        createdMemberIds.forEach(memberRepository::deleteById);
        createdProjectIds.forEach(projectRepository::deleteById);
        createdMemberIds.clear();
        createdProjectIds.clear();
    }

    public ProjectRepository getProjectRepository() {
        return projectRepository;
    }

    public ProjectMemberRepository getMemberRepository() {
        return memberRepository;
    }
}
