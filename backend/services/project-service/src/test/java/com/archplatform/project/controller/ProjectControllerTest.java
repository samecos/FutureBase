package com.archplatform.project.controller;

import com.archplatform.project.dto.CreateProjectRequest;
import com.archplatform.project.dto.ProjectDTO;
import com.archplatform.project.dto.UpdateProjectRequest;
import com.archplatform.project.entity.Project;
import com.archplatform.project.entity.enums.ProjectStatus;
import com.archplatform.project.exception.ProjectNotFoundException;
import com.archplatform.project.security.ProjectSecurity;
import com.archplatform.project.service.ProjectService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.PageImpl;
import org.springframework.data.domain.Pageable;
import org.springframework.http.MediaType;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.web.servlet.MockMvc;

import java.time.LocalDateTime;
import java.util.Collections;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@WebMvcTest(ProjectController.class)
class ProjectControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockBean
    private ProjectService projectService;

    @MockBean
    private ProjectSecurity projectSecurity;

    private UUID projectId;
    private UUID ownerId;
    private Project testProject;
    private CreateProjectRequest createRequest;

    @BeforeEach
    void setUp() {
        projectId = UUID.randomUUID();
        ownerId = UUID.randomUUID();

        testProject = Project.builder()
                .id(projectId)
                .name("Test Project")
                .description("Test Description")
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .createdAt(LocalDateTime.now())
                .updatedAt(LocalDateTime.now())
                .build();

        createRequest = new CreateProjectRequest();
        createRequest.setName("New Project");
        createRequest.setDescription("New Description");
        createRequest.setLocation("New York");
    }

    @Test
    @DisplayName("Should create project successfully")
    @WithMockUser(username = "testuser")
    void createProject_Success() throws Exception {
        when(projectService.createProject(any(CreateProjectRequest.class), any()))
                .thenReturn(testProject);

        mockMvc.perform(post("/api/v1/projects")
                .with(csrf())
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(createRequest)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").value(projectId.toString()))
                .andExpect(jsonPath("$.name").value("Test Project"))
                .andExpect(jsonPath("$.status").value("ACTIVE"));
    }

    @Test
    @DisplayName("Should get project by ID")
    @WithMockUser
    void getProject_Success() throws Exception {
        when(projectService.getProject(projectId)).thenReturn(testProject);
        when(projectSecurity.canAccessProject(any(), eq(projectId))).thenReturn(true);

        mockMvc.perform(get("/api/v1/projects/{id}", projectId))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(projectId.toString()))
                .andExpect(jsonPath("$.name").value("Test Project"));
    }

    @Test
    @DisplayName("Should return not found for non-existent project")
    @WithMockUser
    void getProject_NotFound() throws Exception {
        when(projectService.getProject(projectId))
                .thenThrow(new ProjectNotFoundException("Project not found"));

        mockMvc.perform(get("/api/v1/projects/{id}", projectId))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("PROJECT_NOT_FOUND"));
    }

    @Test
    @DisplayName("Should update project successfully")
    @WithMockUser
    void updateProject_Success() throws Exception {
        UpdateProjectRequest request = new UpdateProjectRequest();
        request.setName("Updated Name");
        request.setDescription("Updated Description");

        Project updatedProject = Project.builder()
                .id(projectId)
                .name("Updated Name")
                .description("Updated Description")
                .ownerId(ownerId)
                .status(ProjectStatus.ACTIVE)
                .build();

        when(projectService.updateProject(eq(projectId), any(UpdateProjectRequest.class), any()))
                .thenReturn(updatedProject);

        mockMvc.perform(put("/api/v1/projects/{id}", projectId)
                .with(csrf())
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.name").value("Updated Name"))
                .andExpect(jsonPath("$.description").value("Updated Description"));
    }

    @Test
    @DisplayName("Should delete project successfully")
    @WithMockUser
    void deleteProject_Success() throws Exception {
        mockMvc.perform(delete("/api/v1/projects/{id}", projectId)
                .with(csrf()))
                .andExpect(status().isNoContent());
    }

    @Test
    @DisplayName("Should archive project successfully")
    @WithMockUser
    void archiveProject_Success() throws Exception {
        Project archivedProject = Project.builder()
                .id(projectId)
                .name("Test Project")
                .status(ProjectStatus.ARCHIVED)
                .build();

        when(projectService.archiveProject(projectId, ownerId)).thenReturn(archivedProject);

        mockMvc.perform(post("/api/v1/projects/{id}/archive", projectId)
                .with(csrf()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("ARCHIVED"));
    }

    @Test
    @DisplayName("Should list projects")
    @WithMockUser
    void listProjects_Success() throws Exception {
        Page<Project> projectPage = new PageImpl<>(Collections.singletonList(testProject));

        when(projectService.listProjects(any(), any(Pageable.class))).thenReturn(projectPage);

        mockMvc.perform(get("/api/v1/projects")
                .param("page", "0")
                .param("size", "10"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.content").isArray())
                .andExpect(jsonPath("$.content[0].id").value(projectId.toString()))
                .andExpect(jsonPath("$.totalElements").value(1));
    }
}
