//go:build windows

package doctor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// platformChecks adiciona as verificacoes que so fazem sentido no Windows:
// caminhos longos exigem opt-in do sistema operacional, arquivos
// somente-nuvem sao um mecanismo de OneDrive/atributo NTFS, e colisao de
// casing so importa porque o sistema de arquivos e insensivel a maiusculas.
// Todas leem de scan em vez de varrer o cofre de novo.
func platformChecks(scan vaultScan) []Result {
	return []Result{
		checkLongPathsEnabled(scan),
		checkCloudOnlyFiles(scan),
		checkCasingCollisions(scan),
	}
}

// checkLongPathsEnabled avisa quando ha um caminho acima do limiar e o
// registro nao tem o opt-in de caminhos longos do Windows ligado. Sem os
// dois, e apenas informativo: caminho curto nao precisa do opt-in, e opt-in
// ligado ja resolve o caminho longo.
func checkLongPathsEnabled(scan vaultScan) Result {
	const name = "caminhos longos habilitados"

	if res, failed := scanStatus(scan, name); failed {
		return res
	}
	if scan.longestPathLen <= longPathThreshold {
		return Result{Name: name, Status: StatusOK}
	}

	if !longPathsEnabled() {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("LongPathsEnabled != 1 no registro e ha caminho de %d caracteres: %s", scan.longestPathLen, scan.longestPath),
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// longPathsEnabled le HKLM\SYSTEM\CurrentControlSet\Control\FileSystem!LongPathsEnabled.
// Qualquer falha ao ler (chave ou valor ausente) e tratada como "desligado":
// e o comportamento padrao do Windows quando o opt-in nunca foi aplicado.
func longPathsEnabled() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\FileSystem`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() { _ = k.Close() }()

	val, _, err := k.GetIntegerValue("LongPathsEnabled")
	if err != nil {
		return false
	}
	return val == 1
}

// checkCloudOnlyFiles avisa quando alguma nota ainda nao foi baixada pelo
// sincronizador de nuvem (OneDrive Files On-Demand e equivalentes). A
// deteccao e por atributo de arquivo (vault.IsCloudOnly, coletado por
// scanVault via vault.Walk) — nunca abre o arquivo, que e exatamente o que
// dispararia a hidratacao que esta verificacao existe para evitar.
func checkCloudOnlyFiles(scan vaultScan) Result {
	const name = "arquivos somente-nuvem"

	if res, failed := scanStatus(scan, name); failed {
		return res
	}
	if scan.cloudOnlyCount > 0 {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d nota(s) ainda nao baixada(s) pelo sincronizador de nuvem", scan.cloudOnlyCount),
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// checkCasingCollisions avisa quando duas notas distintas no disco tem o
// mesmo caminho ao comparar em minusculas. NTFS e insensivel a maiusculas por
// padrao, mas preserva a grafia — duas notas assim colidem de formas sutis em
// qualquer ferramenta que normalize o caminho antes de usar como chave.
func checkCasingCollisions(scan vaultScan) Result {
	const name = "colisoes de casing"

	if res, failed := scanStatus(scan, name); failed {
		return res
	}
	if len(scan.casingCollisions) > 0 {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d colisao(oes): %s", len(scan.casingCollisions), strings.Join(scan.casingCollisions, "; ")),
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// diskFreeBytes consulta o espaco livre do volume que contem path, sem exigir
// que path seja a raiz do volume — GetDiskFreeSpaceEx aceita qualquer
// diretorio e resolve o volume sozinho.
func diskFreeBytes(path string) (uint64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, fmt.Errorf("resolvendo %q: %w", path, err)
	}
	p, err := windows.UTF16PtrFromString(vault.LongPath(abs))
	if err != nil {
		return 0, fmt.Errorf("convertendo %q: %w", abs, err)
	}

	var freeAvailable, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvailable, &total, &totalFree); err != nil {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx(%q): %w", abs, err)
	}
	return freeAvailable, nil
}
