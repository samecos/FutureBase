package com.archplatform.project.integration;

import com.archplatform.project.dto.AddMemberRequest;
import com.archplatform.project.entity.Project;
import com.archplatform.project.entity.enums.ProjectRole;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;

import java.util.UUID;

import static org.hamcrest.Matchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

/**
 * Integration tests for Project Member management.
 */
class ProjectMemberIntegrationTest extends BaseIntegrationTest {

    @Test
    @DisplayName("Should add member to project successfully")
    void addMember_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID newMemberId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);

        AddMemberRequest request = new AddMemberRequest();
        request.setUserId(newMemberId);
        request.setRole(ProjectRole.EDITOR);

        // When & Then
        mockMvc.perform(post("/api/v1/projects/{id}/members", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.userId").value(newMemberId.toString()))
                .andExpect(jsonPath("$.projectId").value(project.getId().toString()))
                .andExpect(jsonPath("$.role").value("EDITOR"));
    }

    @Test
    @DisplayName("Should return forbidden when non-admin tries to add member")
    void addMember_NonAdmin_ReturnsForbidden() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID editorId = UUID.randomUUID();
        UUID newMemberId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);
        testDataHelper.addMember(project.getId(), editorId, ProjectRole.EDITOR);

        AddMemberRequest request = new AddMemberRequest();
        request.setUserId(newMemberId);
        request.setRole(ProjectRole.VIEWER);

        // When & Then
        mockMvc.perform(post("/api/v1/projects/{id}/members", project.getId())
                .header("Authorization", generateMockJwtToken(editorId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isForbidden());
    }

    @Test
    @DisplayName("Should list project members")
    void listMembers_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID member1Id = UUID.randomUUID();
        UUID member2Id = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);
        testDataHelper.addMember(project.getId(), member1Id, ProjectRole.EDITOR);
        testDataHelper.addMember(project.getId(), member2Id, ProjectRole.VIEWER);

        // When & Then
        mockMvc.perform(get("/api/v1/projects/{id}/members", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").isArray())
                .andExpect(jsonPath("$.length()").value(3)) // Owner + 2 members
                .andExpect(jsonPath("$[*].role").value(containsInAnyOrder("OWNER", "EDITOR", "VIEWER")));
    }

    @Test
    @DisplayName("Should update member role successfully")
    void updateMemberRole_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID memberId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);
        testDataHelper.addMember(project.getId(), memberId, ProjectRole.VIEWER);

        // When & Then
        mockMvc.perform(put("/api/v1/projects/{id}/members/{userId}", project.getId(), memberId)
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .param("role", "ADMIN"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.userId").value(memberId.toString()))
                .andExpect(jsonPath("$.role").value("ADMIN"));
    }

    @Test
    @DisplayName("Should remove member from project")
    void removeMember_Success() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID memberId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);
        testDataHelper.addMember(project.getId(), memberId, ProjectRole.EDITOR);

        // When & Then
        mockMvc.perform(delete("/api/v1/projects/{id}/members/{userId}", project.getId(), memberId)
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isNoContent());

        // Verify member is removed
        mockMvc.perform(get("/api/v1/projects/{id}/members", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[*].userId").value(not(contains(memberId.toString()))));
    }

    @Test
    @DisplayName("Should return bad request when trying to remove owner")
    void removeMember_Owner_ReturnsBadRequest() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);

        // When & Then
        mockMvc.perform(delete("/api/v1/projects/{id}/members/{userId}", project.getId(), ownerId)
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser")))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("CANNOT_REMOVE_OWNER"));
    }

    @Test
    @DisplayName("Should return conflict when adding existing member")
    void addMember_ExistingMember_ReturnsConflict() throws Exception {
        // Given
        UUID ownerId = UUID.randomUUID();
        UUID existingMemberId = UUID.randomUUID();
        Project project = testDataHelper.createTestProject("Test Project", "Description", ownerId);
        testDataHelper.addMember(project.getId(), existingMemberId, ProjectRole.EDITOR);

        AddMemberRequest request = new AddMemberRequest();
        request.setUserId(existingMemberId);
        request.setRole(ProjectRole.VIEWER);

        // When & Then
        mockMvc.perform(post("/api/v1/projects/{id}/members", project.getId())
                .header("Authorization", generateMockJwtToken(ownerId.toString(), "testuser"))
                .contentType(MediaType.APPLICATION_JSON)
                .content(asJsonString(request)))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.code").value("MEMBER_ALREADY_EXISTS"));
    }
}
