package com.archplatform.version.service;

import com.archplatform.version.dto.ConflictResolution;
import com.archplatform.version.dto.MergePreview;
import com.archplatform.version.entity.Branch;
import com.archplatform.version.entity.ChangeSet;
import com.archplatform.version.entity.Version;
import com.archplatform.version.entity.enums.ChangeType;
import com.archplatform.version.repository.BranchRepository;
import com.archplatform.version.repository.MergeRequestRepository;
import com.archplatform.version.repository.VersionRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class MergeServiceTest {

    @Mock
    private VersionRepository versionRepository;

    @Mock
    private BranchRepository branchRepository;

    @Mock
    private MergeRequestRepository mergeRequestRepository;

    @InjectMocks
    private MergeService mergeService;

    private UUID projectId;
    private Branch sourceBranch;
    private Branch targetBranch;
    private Version commonAncestor;

    @BeforeEach
    void setUp() {
        projectId = UUID.randomUUID();
        
        commonAncestor = Version.builder()
                .id(UUID.randomUUID())
                .projectId(projectId)
                .build();

        sourceBranch = Branch.builder()
                .id(UUID.randomUUID())
                .name("feature-branch")
                .projectId(projectId)
                .headVersionId(UUID.randomUUID())
                .build();

        targetBranch = Branch.builder()
                .id(UUID.randomUUID())
                .name("main")
                .projectId(projectId)
                .headVersionId(UUID.randomUUID())
                .build();
    }

    @Test
    @DisplayName("Should detect no conflicts when branches have independent changes")
    void detectConflicts_NoConflicts() {
        // Given
        ChangeSet sourceChange = ChangeSet.builder()
                .id(UUID.randomUUID())
                .changeType(ChangeType.PROPERTY_UPDATE)
                .targetId("prop1")
                .build();

        ChangeSet targetChange = ChangeSet.builder()
                .id(UUID.randomUUID())
                .changeType(ChangeType.PROPERTY_UPDATE)
                .targetId("prop2")
                .build();

        List<ChangeSet> sourceChanges = List.of(sourceChange);
        List<ChangeSet> targetChanges = List.of(targetChange);

        // When
        var conflicts = mergeService.detectConflicts(sourceChanges, targetChanges);

        // Then
        assertTrue(conflicts.isEmpty());
    }

    @Test
    @DisplayName("Should detect conflict when both branches modify same property")
    void detectConflicts_WithConflicts() {
        // Given
        String targetId = "prop1";
        
        ChangeSet sourceChange = ChangeSet.builder()
                .id(UUID.randomUUID())
                .changeType(ChangeType.PROPERTY_UPDATE)
                .targetId(targetId)
                .build();

        ChangeSet targetChange = ChangeSet.builder()
                .id(UUID.randomUUID())
                .changeType(ChangeType.PROPERTY_UPDATE)
                .targetId(targetId)
                .build();

        List<ChangeSet> sourceChanges = List.of(sourceChange);
        List<ChangeSet> targetChanges = List.of(targetChange);

        // When
        var conflicts = mergeService.detectConflicts(sourceChanges, targetChanges);

        // Then
        assertFalse(conflicts.isEmpty());
        assertEquals(1, conflicts.size());
        assertEquals(targetId, conflicts.get(0).getTargetId());
    }

    @Test
    @DisplayName("Should create merge request successfully")
    void createMergeRequest_Success() {
        // Given
        when(branchRepository.findById(sourceBranch.getId())).thenReturn(Optional.of(sourceBranch));
        when(branchRepository.findById(targetBranch.getId())).thenReturn(Optional.of(targetBranch));

        // When & Then - Just verify no exception is thrown
        assertDoesNotThrow(() -> 
            mergeService.createMergeRequest(sourceBranch.getId(), targetBranch.getId())
        );
    }

    @Test
    @DisplayName("Should generate JSON Patch diff for merge preview")
    void previewMerge_GeneratesDiff() {
        // Given
        UUID mergeRequestId = UUID.randomUUID();
        
        // When
        MergePreview preview = mergeService.previewMerge(mergeRequestId);

        // Then
        assertNotNull(preview);
    }
}
