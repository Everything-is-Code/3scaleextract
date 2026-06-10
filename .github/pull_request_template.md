## Proposed change

<!-- Qué cambia y por qué. Comunicá el objetivo al reviewer/PO. -->

## Type of change

<!-- Marcá **una** categoría principal -->

- [ ] Bugfix (non-breaking)
- [ ] New feature
- [ ] Tests / CI
- [ ] Documentation
- [ ] Integration cross-repo (3scaleextract ↔ gateforge)
- [ ] Breaking change

## Breaking change

<!-- Solo si aplica: qué rompe, cómo migrar, por qué lo hicimos. Si no aplica, borrá esta sección. -->

## Checklist RHCL

- [ ] Tests agregados o actualizados (`go test ./...` o `mvn test` / `npm test`)
- [ ] CI no introduce `-DskipTests` sin justificación
- [ ] Sin secretos en el diff (tokens, kubeconfigs, OIDC secrets)
- [ ] Docs [rhcl-ai](https://github.com/Everything-is-Code/rhcl-ai) actualizados si cambia contrato export/import
- [ ] Probado en lab local si aplica (seed → export → analyze)

## Test plan

<!-- Pasos concretos para validar manualmente -->

## Additional information

- Fixes / Closes: <!-- EXT-*, GF-*, INT-* o #número -->
- Related: <!-- cross-repo links -->
- Milestone: <!-- M1 / M2 / M3 -->
