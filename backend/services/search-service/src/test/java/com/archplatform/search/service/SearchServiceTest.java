package com.archplatform.search.service;

import co.elastic.clients.elasticsearch.ElasticsearchClient;
import co.elastic.clients.elasticsearch.core.SearchRequest;
import co.elastic.clients.elasticsearch.core.SearchResponse;
import com.archplatform.search.dto.SearchRequestDTO;
import com.archplatform.search.dto.SearchResult;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.io.IOException;
import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class SearchServiceTest {

    @Mock
    private ElasticsearchClient elasticsearchClient;

    @Mock
    private SearchResponse<Object> searchResponse;

    private SearchService searchService;

    @BeforeEach
    void setUp() {
        searchService = new SearchService(elasticsearchClient);
    }

    @Test
    @DisplayName("Should search projects by keyword")
    void searchProjects_Success() throws IOException {
        // Given
        SearchRequestDTO request = new SearchRequestDTO();
        request.setQuery("building design");
        request.setType("project");
        request.setPage(0);
        request.setSize(10);

        when(elasticsearchClient.search(any(SearchRequest.class), any(Class.class)))
                .thenReturn(searchResponse);
        when(searchResponse.hits()).thenReturn(null); // Simplified for test

        // When
        SearchResult result = searchService.search(request);

        // Then
        assertNotNull(result);
    }

    @Test
    @DisplayName("Should provide search suggestions")
    void suggest_Success() throws IOException {
        // Given
        String query = "archi";

        // When
        List<String> suggestions = searchService.suggest(query);

        // Then
        assertNotNull(suggestions);
    }

    @Test
    @DisplayName("Should index project document")
    void indexProject_Success() throws IOException {
        // Given
        UUID projectId = UUID.randomUUID();
        String projectName = "Test Project";
        String description = "A test project description";

        // When & Then
        assertDoesNotThrow(() -> 
            searchService.indexProject(projectId, projectName, description)
        );
    }

    @Test
    @DisplayName("Should delete project from index")
    void deleteProject_Success() throws IOException {
        // Given
        UUID projectId = UUID.randomUUID();

        // When & Then
        assertDoesNotThrow(() -> searchService.deleteProject(projectId));
    }
}
