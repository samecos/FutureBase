package com.archplatform.project.controller;

import com.archplatform.project.dto.AddMemberRequest;
import com.archplatform.project.dto.ProjectMemberDTO;
import com.archplatform.project.entity.ProjectMember;
import com.archplatform.project.entity.enums.ProjectRole;
import com.archplatform.project.service.ProjectMemberService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.web.servlet.MockMvc;

import java.time.LocalDateTime;
import java.util.Arrays;
import java.util.List;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@WebMvcTest(ProjectMemberController.class)
class ProjectMemberControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockBean
    private ProjectMemberService memberService;

    private UUID projectId;
    private UUID userId;
    private ProjectMember testMember;

    @BeforeEach
    void setUp() {
        projectId = UUID.randomUUID();
        userId = UUID.randomUUID();

        testMember = ProjectMember.builder()
                .id(UUID.randomUUID())
                .projectId(projectId)
                .userId(userId)
                .role(ProjectRole.EDITOR)
                .joinedAt(LocalDateTime.now())
                .build();
    }

    @Test
    @DisplayName("Should add member to project")
    @WithMockUser
    void addMember_Success() throws Exception {
        AddMemberRequest request = new AddMemberRequest();
        request.setUserId(userId);
        request.setRole(ProjectRole.EDITOR);

        when(memberService.addMember(eq(projectId), any(AddMemberRequest.class), any()))
                .thenReturn(testMember);

        mockMvc.perform(post("/api/v1/projects/{id}/members", projectId)
                .with(csrf())
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.userId").value(userId.toString()))
                .andExpect(jsonPath("$.role").value("EDITOR"));
    }

    @Test
    @DisplayName("Should list project members")
    @WithMockUser
    void listMembers_Success() throws Exception {
        ProjectMember member1 = ProjectMember.builder()
                .id(UUID.randomUUID())
                .projectId(projectId)
                .userId(UUID.randomUUID())
                .role(ProjectRole.OWNER)
                .build();

        ProjectMember member2 = ProjectMember.builder()
                .id(UUID.randomUUID())
                .projectId(projectId)
                .userId(UUID.randomUUID())
                .role(ProjectRole.EDITOR)
                .build();

        List<ProjectMember> members = Arrays.asList(member1, member2);
        when(memberService.getProjectMembers(projectId)).thenReturn(members);

        mockMvc.perform(get("/api/v1/projects/{id}/members", projectId))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").isArray())
                .andExpect(jsonPath("$.length()").value(2))
                .andExpect(jsonPath("$[0].role").value("OWNER"))
                .andExpect(jsonPath("$[1].role").value("EDITOR"));
    }

    @Test
    @DisplayName("Should update member role")
    @WithMockUser
    void updateMemberRole_Success() throws Exception {
        ProjectMember updatedMember = ProjectMember.builder()
                .id(testMember.getId())
                .projectId(projectId)
                .userId(userId)
                .role(ProjectRole.ADMIN)
                .build();

        when(memberService.updateMemberRole(eq(projectId), eq(userId), eq(ProjectRole.ADMIN), any()))
                .thenReturn(updatedMember);

        mockMvc.perform(put("/api/v1/projects/{id}/members/{userId}", projectId, userId)
                .with(csrf())
                .param("role", "ADMIN"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.role").value("ADMIN"));
    }

    @Test
    @DisplayName("Should remove member from project")
    @WithMockUser
    void removeMember_Success() throws Exception {
        mockMvc.perform(delete("/api/v1/projects/{id}/members/{userId}", projectId, userId)
                .with(csrf()))
                .andExpect(status().isNoContent());
    }

    @Test
    @DisplayName("Should return bad request when trying to remove owner")
    @WithMockUser
    void removeMember_Owner_ReturnsBadRequest() throws Exception {
        when(memberService.removeMember(eq(projectId), eq(userId), any()))
                .thenThrow(new IllegalArgumentException("Cannot remove owner"));

        mockMvc.perform(delete("/api/v1/projects/{id}/members/{userId}", projectId, userId)
                .with(csrf()))
                .andExpect(status().isBadRequest());
    }
}
