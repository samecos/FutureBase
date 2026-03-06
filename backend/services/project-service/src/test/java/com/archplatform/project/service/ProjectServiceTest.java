package com.archplatform.project.service;

import com.archplatform.project.dto.CreateProjectRequest;
import com.archplatform.project.entity.Project;
import com.archplatform.project.entity.enums.ProjectStatus;
import com.archplatform.project.exception.ProjectNotFoundException;
import com.archplatform.project.repository.ProjectRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class ProjectServiceTest {

    @Mock
    private ProjectRepository projectRepository;

    @InjectMocks
    private ProjectService projectService;

    private UUID ownerId;
    private Project testProject;

    @BeforeEach
    void setUp() {
        ownerId = UUID.randomUUID();
        
        testProject = Project.builder()
                .id(UUID.randomUUID())
                .name("Test Project")
                .description("Test Description")
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .build();
    }

    @Test
    @DisplayName("Should create project successfully")
    void createProject_Success() {
        // Given
        CreateProjectRequest request = new CreateProjectRequest();
        request.setName("Test Project");
        request.setDescription("Test Description");

        when(projectRepository.save(any(Project.class))).thenReturn(testProject);

        // When
        Project result = projectService.createProject(request, ownerId);

        // Then
        assertNotNull(result);
        assertEquals("Test Project", result.getName());
        assertEquals(ownerId, result.getOwnerId());
        verify(projectRepository).save(any(Project.class));
    }

    @Test
    @DisplayName("Should get project by id")
    void getProject_Success() {
        // Given
        UUID projectId = testProject.getId();
        when(projectRepository.findById(projectId)).thenReturn(Optional.of(testProject));

        // When
        Project result = projectService.getProject(projectId);

        // Then
        assertNotNull(result);
        assertEquals(testProject.getName(), result.getName());
    }

    @Test
    @DisplayName("Should throw exception when project not found")
    void getProject_NotFound_ThrowsException() {
        // Given
        UUID projectId = UUID.randomUUID();
        when(projectRepository.findById(projectId)).thenReturn(Optional.empty());

        // When & Then
        assertThrows(ProjectNotFoundException.class, () -> projectService.getProject(projectId));
    }

    @Test
    @DisplayName("Should archive project")
    void archiveProject_Success() {
        // Given
        UUID projectId = testProject.getId();
        when(projectRepository.findById(projectId)).thenReturn(Optional.of(testProject));
        when(projectRepository.save(any(Project.class))).thenReturn(testProject);

        // When
        Project result = projectService.archive(projectId);

        // Then
        assertEquals(ProjectStatus.ARCHIVED, result.getStatus());
    }
}
