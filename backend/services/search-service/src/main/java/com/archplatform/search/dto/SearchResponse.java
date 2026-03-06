package com.archplatform.search.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.Map;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class SearchResponse {

    private List<SearchHit> hits;
    private long totalHits;
    private int totalPages;
    private int currentPage;
    private long took;
    private boolean timedOut;
    private Map<String, Long> aggregations;
    private List<String> suggestions;

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class SearchHit {
        private String id;
        private String index;
        private String type;
        private String title;
        private String description;
        private Map<String, Object> source;
        private Map<String, List<String>> highlights;
        private Double score;
    }
}
