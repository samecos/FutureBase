package com.archplatform.user.repository;

import com.archplatform.user.entity.Permission;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.UUID;

@Repository
public interface PermissionRepository extends JpaRepository<Permission, UUID> {

    Optional<Permission> findByResourceAndAction(String resource, String action);

    @Query("SELECT p FROM Permission p WHERE p.resource = :resource")
    List<Permission> findAllByResource(@Param("resource") String resource);

    @Query("SELECT p FROM Permission p JOIN p.roles r WHERE r.id = :roleId")
    Set<Permission> findAllByRoleId(@Param("roleId") UUID roleId);

    @Query("SELECT p FROM Permission p JOIN p.roles r JOIN r.users u WHERE u.id = :userId")
    Set<Permission> findAllByUserId(@Param("userId") UUID userId);

    @Query("SELECT DISTINCT p.resource FROM Permission p")
    List<String> findAllResources();

    boolean existsByResourceAndAction(String resource, String action);
}
