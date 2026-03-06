package com.archplatform.project.repository;

import com.archplatform.project.entity.Project;
import com.archplatform.project.entity.enums.ProjectStatus;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.orm.jpa.DataJpaTest;
import org.springframework.boot.test.autoconfigure.orm.jpa.TestEntityManager;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Pageable;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@DataJpaTest
class ProjectRepositoryTest {

    @Autowired
    private TestEntityManager entityManager;

    @Autowired
    private ProjectRepository projectRepository;

    private UUID ownerId;

    @BeforeEach
    void setUp() {
        ownerId = UUID.randomUUID();
    }

    @Test
    @DisplayName("Should save and retrieve project by ID")
    void saveAndFindById_Success() {
        // Given
        Project project = Project.builder()
                .id(UUID.randomUUID())
                .name("Test Project")
                .description("Test Description")
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();

        // When
        Project saved = entityManager.persistAndFlush(project);
        Optional<Project> found = projectRepository.findById(saved.getId());

        // Then
        assertTrue(found.isPresent());
        assertEquals("Test Project", found.get().getName());
        assertEquals(ownerId, found.get().getOwnerId());
    }

    @Test
    @DisplayName("Should find projects by owner ID")
    void findByOwnerId_Success() {
        // Given
        Project project1 = createProject("Project 1", ownerId);
        Project project2 = createProject("Project 2", ownerId);
        createProject("Other Project", UUID.randomUUID()); // Different owner

        entityManager.persistAndFlush(project1);
        entityManager.persistAndFlush(project2);

        // When
        Pageable pageable = PageRequest.of(0, 10);
        Page<Project> projects = projectRepository.findByOwnerId(ownerId, pageable);

        // Then
        assertEquals(2, projects.getTotalElements());
        assertTrue(projects.getContent().stream()
                .allMatch(p -> p.getOwnerId().equals(ownerId)));
    }

    @Test
    @DisplayName("Should find projects by owner and status")
    void findByOwnerIdAndStatus_Success() {
        // Given
        Project activeProject = createProject("Active Project", ownerId);
        Project archivedProject = Project.builder()
                .name("Archived Project")
                .ownerId(ownerId)
                .status(ProjectStatus.ARCHIVED)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();

        entityManager.persistAndFlush(activeProject);
        entityManager.persistAndFlush(archivedProject);

        // When
        Pageable pageable = PageRequest.of(0, 10);
        Page<Project> activeProjects = projectRepository.findByOwnerIdAndStatus(ownerId, ProjectStatus.ACTIVE, pageable);

        // Then
        assertEquals(1, activeProjects.getTotalElements());
        assertEquals(ProjectStatus.ACTIVE, activeProjects.getContent().get(0).getStatus());
    }

    @Test
    @DisplayName("Should find project by ID and owner")
    void findByIdAndOwnerId_Success() {
        // Given
        Project project = createProject("Test Project", ownerId);
        Project saved = entityManager.persistAndFlush(project);

        // When
        Optional<Project> found = projectRepository.findByIdAndOwnerId(saved.getId(), ownerId);
        Optional<Project> notFound = projectRepository.findByIdAndOwnerId(saved.getId(), UUID.randomUUID());

        // Then
        assertTrue(found.isPresent());
        assertFalse(notFound.isPresent());
    }

    @Test
    @DisplayName("Should check if project name exists for owner")
    void existsByNameAndOwnerId_Success() {
        // Given
        Project project = createProject("Unique Name", ownerId);
        entityManager.persistAndFlush(project);

        // When & Then
        assertTrue(projectRepository.existsByNameAndOwnerId("Unique Name", ownerId));
        assertFalse(projectRepository.existsByNameAndOwnerId("Different Name", ownerId));
        assertFalse(projectRepository.existsByNameAndOwnerId("Unique Name", UUID.randomUUID()));
    }

    @Test
    @DisplayName("Should update project")
    void updateProject_Success() {
        // Given
        Project project = createProject("Original Name", ownerId);
        Project saved = entityManager.persistAndFlush(project);

        // When
        saved.setName("Updated Name");
        saved.setDescription("Updated Description");
        projectRepository.save(saved);
        entityManager.flush();

        // Then
        Optional<Project> updated = projectRepository.findById(saved.getId());
        assertTrue(updated.isPresent());
        assertEquals("Updated Name", updated.get().getName());
        assertEquals("Updated Description", updated.get().getDescription());
    }

    @Test
    @DisplayName("Should soft delete project")
    void softDeleteProject_Success() {
        // Given
        Project project = createProject("To Delete", ownerId);
        Project saved = entityManager.persistAndFlush(project);

        // When
        projectRepository.delete(saved);
        entityManager.flush();

        // Then
        Optional<Project> found = projectRepository.findById(saved.getId());
        assertFalse(found.isPresent());
    }

    @Test
    @DisplayName("Should count projects by owner")
    void countByOwnerId_Success() {
        // Given
        entityManager.persistAndFlush(createProject("Project 1", ownerId));
        entityManager.persistAndFlush(createProject("Project 2", ownerId));
        entityManager.persistAndFlush(createProject("Other", UUID.randomUUID()));

        // When
        long count = projectRepository.countByOwnerId(ownerId);

        // Then
        assertEquals(2, count);
    }

    @Test
    @DisplayName("Should search projects by name")
    void findByNameContainingIgnoreCase_Success() {
        // Given
        entityManager.persistAndFlush(createProject("Architecture Design", ownerId));
        entityManager.persistAndFlush(createProject("Building Plan", ownerId));
        entityManager.persistAndFlush(createProject("Other Project", ownerId));

        // When
        Pageable pageable = PageRequest.of(0, 10);
        Page<Project> results = projectRepository.findByNameContainingIgnoreCaseAndOwnerId("arch", ownerId, pageable);

        // Then
        assertEquals(1, results.getTotalElements());
        assertEquals("Architecture Design", results.getContent().get(0).getName());
    }

    private Project createProject(String name, UUID ownerId) {
        return Project.builder()
                .name(name)
                .description("Description for " + name)
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();
    }
}
