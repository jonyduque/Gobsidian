//go:build windows

package doctor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jonyd/gobsidian/internal/config"
	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// platformChecks adiciona as verificacoes que so fazem sentido no Windows:
// caminhos longos exigem opt-in do sistema operacional, arquivos
// somente-nuvem sao um mecanismo de OneDrive/atributo NTFS, e colisao de
// casing so importa porque o sistema de arquivos e insensivel a maiusculas.
func platformChecks() []check {
	return []check{
		checkLongPathsEnabled,
		checkCloudOnlyFiles,
		checkCasingCollisions,
	}
}

// checkLongPathsEnabled avisa quando ha um caminho acima do limiar e o
// registro nao tem o opt-in de caminhos longos do Windows ligado. Sem os
// dois, e apenas informativo: caminho curto nao precisa do opt-in, e opt-in
// ligado ja resolve o caminho longo.
func checkLongPathsEnabled(ctx context.Context, cfg config.Config) Result {
	const name = "caminhos longos habilitados"

	length, longest, err := longestVaultPath(ctx, cfg)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}
	if length <= longPathThreshold {
		return Result{Name: name, Status: StatusOK}
	}

	if !longPathsEnabled() {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("LongPathsEnabled != 1 no registro e ha caminho de %d caracteres: %s", length, longest),
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
	defer k.Close()

	val, _, err := k.GetIntegerValue("LongPathsEnabled")
	if err != nil {
		return false
	}
	return val == 1
}

// checkCloudOnlyFiles avisa quando alguma nota ainda nao foi baixada pelo
// sincronizador de nuvem (OneDrive Files On-Demand e equivalentes). A
// deteccao e por atributo de arquivo (vault.IsCloudOnly, via vault.Walk) —
// nunca abre o arquivo, que e exatamente o que dispararia a hidratacao que
// esta verificacao existe para evitar.
func checkCloudOnlyFiles(ctx context.Context, cfg config.Config) Result {
	const name = "arquivos somente-nuvem"

	var count int
	err := walkVault(ctx, cfg, func(e vault.Entry) {
		if e.IsNote && e.CloudOnly {
			count++
		}
	})
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}
	if count > 0 {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d nota(s) ainda nao baixada(s) pelo sincronizador de nuvem", count),
		}
	}
	return Result{Name: name, Status: StatusOK}
}

// checkCasingCollisions avisa quando duas notas distintas no disco tem o
// mesmo caminho ao comparar em minusculas. NTFS e insensivel a maiusculas por
// padrao, mas preserva a grafia — duas notas assim colidem de formas sutis em
// qualquer ferramenta que normalize o caminho antes de usar como chave.
func checkCasingCollisions(ctx context.Context, cfg config.Config) Result {
	const name = "colisoes de casing"

	seen := make(map[string]string)
	var collisions []string
	err := walkVault(ctx, cfg, func(e vault.Entry) {
		if !e.IsNote {
			return
		}
		key := strings.ToLower(string(e.Path))
		if prev, ok := seen[key]; ok {
			if prev != string(e.Path) {
				collisions = append(collisions, fmt.Sprintf("%s <-> %s", prev, e.Path))
			}
			return
		}
		seen[key] = string(e.Path)
	})
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: fmt.Sprintf("varredura interrompida: %v", err)}
	}
	if len(collisions) > 0 {
		return Result{
			Name:   name,
			Status: StatusWarn,
			Detail: fmt.Sprintf("%d colisao(oes): %s", len(collisions), strings.Join(collisions, "; ")),
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
