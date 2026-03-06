package com.archplatform.search.service;

import co.elastic.clients.elasticsearch.ElasticsearchClient;
import com.archplatform.search.dto.SearchRequest;
import com.archplatform.search.dto.SearchResponse;
import com.archplatform.search.dto.SearchResponse.SearchHit;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class SearchService {

    private final ElasticsearchClient elasticsearchClient;

    public SearchResponse search(SearchRequest request) throws IOException {
        log.info("Search query: {}", request.getQuery());

        // Return empty response for now
        return SearchResponse.builder()
            .hits(Collections.emptyList())
            .totalHits(0)
            .totalPages(0)
            .currentPage(request.getPage())
            .took(0)
            .timedOut(false)
            .build();
    }

    public List<String> getSuggestions(String query, String index) throws IOException {
        log.info("Suggestions query: {}, index: {}", query, index);
        return Collections.emptyList();
    }

    public Map<String, Long> getAggregations(String field, String index) throws IOException {
        log.info("Aggregations field: {}, index: {}", field, index);
        return Collections.emptyMap();
    }

    private SearchHit mapToSearchHit(Hit<Map> hit) {
        Map<String, Object> source = hit.source();

        Map<String, List<String>> highlights = new HashMap<>();
}
