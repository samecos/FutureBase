package com.archplatform.project.integration;

import com.archplatform.project.dto.CreateProjectRequest;
import com.archplatform.project.dto.UpdateProjectRequest;
import com.archplatform.project.entity.Project;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

import java.util.UUID;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

/**
 * Integration tests for Project CRUD operations.
 */
class ProjectCrudIntegrationTest extends BaseIntegrationTest {

    @Test
    @DisplayName("Should create project successfully")
    void createProject_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        CreateProjectRequest request = new CreateProjectRequest();
        request.setName("New Project");
        request.setDescription("Project Description");
        request.setLocation("New York");

        // When & Then
        mockMvc.perform(post("/api/v1/projects")
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.id").exists())
                .andExpect(jsonPath("$.name").value("New Project"))
                .andExpect(jsonPath("$.description").value("Project Description"))
                .andExpect(jsonPath("$.location").value("New York"))
                .andExpect(jsonPath("$.ownerId").value(ownerId.toString()))
                .andExpect(jsonPath("$.status").value("ACTIVE"))
                .andExpect(jsonPath("$.createdAt").exists())
                .andExpect(jsonPath("$.updatedAt").exists());
    }

    @Test
    @DisplayName("Should return bad request when project name is blank")
    void createProject_BlankName_ReturnsBadRequest() throws Exception {
        // Given
        CreateProjectRequest request = new CreateProjectRequest();
        request.setName("");
        request.setDescription("Description");

        // When & Then
        mockMvc.perform(post("/api/v1/projects")
                .header("Authorization", generateMockJwtToken(UUID.randomUUID().toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("VALIDATION_ERROR"));
    }

    @Test
    @DisplayName("Should get project by ID")
    void getProject_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);

        // When & Then
        mockMvc.perform(get("/api/v1/projects/{id}", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(project.getId().toString()))
                .andExpect(jsonPath("$.name").value("Test Project"))
                .andExpect(jsonPath("$.description").value("Description"));
    }

    @Test
    @DisplayName("Should return not found for non-existent project")
    void getProject_NotFound_Returns404() throws Exception {
        // Given
        UUID nonExistentId = UUID.randomUUID();
        UUID userId = UUID.randomUUID();

        // When & Then
        mockMvc.perform(get("/api/v1/projects/{id}", nonExistentId)
                .header("Authorization", generateMockJwtToken(userId.toString(), "testuser")))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("PROJECT_NOT_FOUND"));
    }

    @Test
    @DisplayName("Should update project successfully")
    void updateProject_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Original Name", "Original Description", ownerId);

        UpdateProjectRequest request = new UpdateProjectRequest();
        request.setName("Updated Name");
        request.setDescription("Updated Description");

        // When & Then
        mockMvc.perform(put("/api/v1/projects/{id}", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(project.getId().toString()))
                .andExpect(jsonPath("$.name").value("Updated Name"))
                .andExpect(jsonPath("$.description").value("Updated Description"));
    }

    @Test
    @DisplayName("Should return forbidden when non-owner tries to update")
    void updateProject_NonOwner_ReturnsForbidden() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID otherUserId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Original Name", "Description", ownerId);

        UpdateProjectRequest request = new UpdateProjectRequest();
        request.setName("Updated Name");

        // When & Then
        mockMvc.perform(put("/api/v1/projects/{id}", project.getId())
                .header("Authorization", generateMockJwtToken(otherUserId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isForbidden());
    }

    @Test
    @DisplayName("Should delete project successfully")
    void deleteProject_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("To Delete", "Description", ownerId);

        // When & Then
        mockMvc.perform(delete("/api/v1/projects/{id}", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isNoContent());

        // Verify project is soft deleted
        mockMvc.perform(get("/api/v1/projects/{id}", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isNotFound());
    }

    @Test
    @DisplayName("Should archive project successfully")
    void archiveProject_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("To Archive", "Description", ownerId);

        // When & Then
        mockMvc.perform(post("/api/v1/projects/{id}/archive", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(project.getId().toString()))
                .andExpect(jsonPath("$.status").value("ARCHIVED"));
    }

    @Test
    @DisplayName("Should list user's projects")
    void listProjects_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        testDataHelper.createTestProject("Project 1", "Desc 1", ownerId);
        testDataHelper.createTestProject("Project 2", "Desc 2", ownerId);
        testDataHelper.createTestProject("Project 3", "Desc 3", UUID.randomUUID()); // Different owner

        // When & Then
        mockMvc.perform(get("/api/v1/projects")
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .param("page", "0")
                .param("size", "10"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.content").isArray())
                .andExpect(jsonPath("$.content.length()").value(2))
                .andExpect(jsonPath("$.totalElements").value(2));
    }
}
