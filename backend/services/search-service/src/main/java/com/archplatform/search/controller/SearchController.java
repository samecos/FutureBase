package com.archplatform.search.controller;

import com.archplatform.search.dto.SearchRequest;
import com.archplatform.search.dto.SearchResponse;
import com.archplatform.search.service.SearchService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.io.IOException;
import java.util.List;
import java.util.Map;

@Slf4j
@RestController
@RequestMapping("/search")
@RequiredArgsConstructor
public class SearchController {

    private final SearchService searchService;

    @PostMapping
    public ResponseEntity<SearchResponse> search(@Valid @RequestBody SearchRequest request) {
        try {
            SearchResponse response = searchService.search(request);
            return ResponseEntity.ok(response);
        } catch (IOException e) {
            log.error("Search failed", e);
            return ResponseEntity.internalServerError().build();
        }
    }

    @GetMapping
    public ResponseEntity<SearchResponse> searchGet(
            @RequestParam String query,
            @RequestParam(required = false) List<String> indices,
            @RequestParam(required = false, defaultValue = "0") Integer page,
            @RequestParam(required = false, defaultValue = "20") Integer size) {
        
        SearchRequest request = SearchRequest.builder()
            .query(query)
            .indices(indices)
            .page(page)
            .size(size)
            .highlight(true)
            .build();

        try {
            SearchResponse response = searchService.search(request);
            return ResponseEntity.ok(response);
        } catch (IOException e) {
            log.error("Search failed", e);
            return ResponseEntity.internalServerError().build();
        }
    }

    @GetMapping("/suggestions")
    public ResponseEntity<List<String>> getSuggestions(
            @RequestParam String query,
            @RequestParam(required = false) String index) {
        try {
            List<String> suggestions = searchService.getSuggestions(query, index);
            return ResponseEntity.ok(suggestions);
        } catch (IOException e) {
            log.error("Suggestions failed", e);
            return ResponseEntity.internalServerError().build();
        }
    }

    @GetMapping("/aggregations/{field}")
    public ResponseEntity<Map<String, Long>> getAggregations(
            @PathVariable String field,
            @RequestParam(required = false) String index) {
        try {
            Map<String, Long> aggregations = searchService.getAggregations(field, index);
            return ResponseEntity.ok(aggregations);
        } catch (IOException e) {
            log.error("Aggregations failed", e);
            return ResponseEntity.internalServerError().build();
        }
    }

    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "UP"));
    }
}
