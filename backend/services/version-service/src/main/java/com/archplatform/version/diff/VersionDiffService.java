package com.archplatform.version.diff;

import com.archplatform.version.dto.VersionDiffDTO;
import com.archplatform.version.entity.ChangeSet;
import com.archplatform.version.repository.ChangeSetRepository;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.flipkart.zjsonpatch.JsonDiff;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class VersionDiffService {

    private final ChangeSetRepository changeSetRepository;
    private final ObjectMapper objectMapper;

    public VersionDiffDTO compareVersions(UUID sourceVersionId, UUID targetVersionId) {
        // Get change sets between versions
        List<ChangeSet> sourceChanges = changeSetRepository.findAllByVersionId(sourceVersionId);
        List<ChangeSet> targetChanges = changeSetRepository.findAllByVersionId(targetVersionId);

        List<VersionDiffDTO.ChangeDiff> diffs = new ArrayList<>();
        int addedCount = 0;
        int modifiedCount = 0;
        int deletedCount = 0;

        // Analyze changes
        for (ChangeSet change : targetChanges) {
            VersionDiffDTO.ChangeDiff diff = mapToChangeDiff(change);
            diffs.add(diff);

            switch (change.getChangeType()) {
                case CREATE, GEOMETRY_ADDED, ELEMENT_ADDED, LAYER_ADDED -> addedCount++;
                case UPDATE, GEOMETRY_MODIFIED, ELEMENT_MODIFIED, PROPERTY_CHANGED -> modifiedCount++;
                case DELETE, GEOMETRY_DELETED, ELEMENT_REMOVED, LAYER_REMOVED -> deletedCount++;
            }
        }

        return VersionDiffDTO.builder()
            .sourceVersionId(sourceVersionId)
            .targetVersionId(targetVersionId)
            .changes(diffs)
            .addedCount(addedCount)
            .modifiedCount(modifiedCount)
            .deletedCount(deletedCount)
            .build();
    }

    public List<VersionDiffDTO.ChangeDiff> computeJsonDiff(String oldJson, String newJson) {
        List<VersionDiffDTO.ChangeDiff> diffs = new ArrayList<>();

        try {
            JsonNode oldNode = oldJson != null ? objectMapper.readTree(oldJson) : objectMapper.createObjectNode();
            JsonNode newNode = newJson != null ? objectMapper.readTree(newJson) : objectMapper.createObjectNode();

            JsonNode patch = JsonDiff.asJson(oldNode, newNode);

            for (JsonNode operation : patch) {
                String op = operation.get("op").asText();
                String path = operation.get("path").asText();
                
                VersionDiffDTO.ChangeDiff.ChangeDiffBuilder diffBuilder = VersionDiffDTO.ChangeDiff.builder()
                    .operation(mapOperation(op))
                    .path(path);

                if (operation.has("value")) {
                    diffBuilder.newValue(operation.get("value"));
                }
                if (operation.has("oldValue")) {
                    diffBuilder.oldValue(operation.get("oldValue"));
                }

                diffs.add(diffBuilder.build());
            }
        } catch (Exception e) {
            log.error("Failed to compute JSON diff", e);
        }

        return diffs;
    }

    public boolean hasConflicts(UUID sourceVersionId, UUID targetVersionId) {
        // Simplified conflict detection
        // In a real implementation, this would analyze overlapping changes
        List<ChangeSet> sourceChanges = changeSetRepository.findAllByVersionId(sourceVersionId);
        List<ChangeSet> targetChanges = changeSetRepository.findAllByVersionId(targetVersionId);

        // Check for changes to the same entities
        for (ChangeSet sourceChange : sourceChanges) {
            for (ChangeSet targetChange : targetChanges) {
                if (isConflictingChange(sourceChange, targetChange)) {
                    return true;
                }
            }
        }

        return false;
    }

    public int countConflicts(UUID sourceVersionId, UUID targetVersionId) {
        int conflictCount = 0;
        List<ChangeSet> sourceChanges = changeSetRepository.findAllByVersionId(sourceVersionId);
        List<ChangeSet> targetChanges = changeSetRepository.findAllByVersionId(targetVersionId);

        for (ChangeSet sourceChange : sourceChanges) {
            for (ChangeSet targetChange : targetChanges) {
                if (isConflictingChange(sourceChange, targetChange)) {
                    conflictCount++;
                }
            }
        }

        return conflictCount;
    }

    private boolean isConflictingChange(ChangeSet change1, ChangeSet change2) {
        // Same entity modified in both versions
        if (change1.getEntityId() != null && change1.getEntityId().equals(change2.getEntityId())) {
            // Same property modified
            if (change1.getPropertyName() != null && change1.getPropertyName().equals(change2.getPropertyName())) {
                return true;
            }
            // Both modified geometry
            if (isGeometryChange(change1.getChangeType()) && isGeometryChange(change2.getChangeType())) {
                return true;
            }
        }
        return false;
    }

    private boolean isGeometryChange(ChangeSet.ChangeType type) {
        return type == ChangeSet.ChangeType.GEOMETRY_ADDED ||
               type == ChangeSet.ChangeType.GEOMETRY_MODIFIED ||
               type == ChangeSet.ChangeType.GEOMETRY_DELETED;
    }

    private VersionDiffDTO.ChangeDiff mapToChangeDiff(ChangeSet change) {
        return VersionDiffDTO.ChangeDiff.builder()
            .operation(mapChangeTypeToOperation(change.getChangeType()))
            .entityType(change.getEntityType())
            .entityId(change.getEntityId())
            .oldValue(change.getOldValue())
            .newValue(change.getNewValue())
            .build();
    }

    private String mapChangeTypeToOperation(ChangeSet.ChangeType type) {
        return switch (type) {
            case CREATE, GEOMETRY_ADDED, ELEMENT_ADDED, LAYER_ADDED -> "add";
            case UPDATE, GEOMETRY_MODIFIED, ELEMENT_MODIFIED, PROPERTY_CHANGED -> "replace";
            case DELETE, GEOMETRY_DELETED, ELEMENT_REMOVED, LAYER_REMOVED -> "remove";
        };
    }

    private String mapOperation(String jsonPatchOp) {
        return switch (jsonPatchOp) {
            case "add" -> "add";
            case "remove" -> "remove";
            case "replace" -> "replace";
            case "move" -> "move";
            case "copy" -> "copy";
            default -> "unknown";
        };
    }
}
