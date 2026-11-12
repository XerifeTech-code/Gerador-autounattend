package main

// mensagens do motor (avisos e erros) em pt-BR, en e es. A chave é o texto em português.
var msgs = map[string][2]string{
	"Chave do produto inválida: use o formato XXXXX-XXXXX-XXXXX-XXXXX-XXXXX":                       {"Invalid product key: use the format XXXXX-XXXXX-XXXXX-XXXXX-XXXXX", "Clave de producto no válida: use el formato XXXXX-XXXXX-XXXXX-XXXXX-XXXXX"},
	"Nome do computador inválido: até 15 caracteres (letras, números e hífen), sem ser só números": {"Invalid computer name: up to 15 characters (letters, numbers and hyphen), not only digits", "Nombre de equipo no válido: hasta 15 caracteres (letras, números y guion), sin ser solo números"},
	"Informe o nome do usuário local": {"Enter the local user name", "Indique el nombre del usuario local"},
	"Nome de usuário inválido: até 20 caracteres e sem / \\ [ ] : ; | = , + * ? < > \" @": {"Invalid user name: up to 20 characters and none of / \\ [ ] : ; | = , + * ? < > \" @", "Nombre de usuario no válido: hasta 20 caracteres y sin / \\ [ ] : ; | = , + * ? < > \" @"},
	"O nome %q é reservado pelo Windows; escolha outro":                                   {"The name %q is reserved by Windows; choose another one", "El nombre %q está reservado por Windows; elija otro"},
	"Quantidade de logons automáticos deve estar entre 1 e 999":                           {"The number of automatic logons must be between 1 and 999", "La cantidad de inicios de sesión automáticos debe estar entre 1 y 999"},
	"Número do disco inválido":                                                            {"Invalid disk number", "Número de disco no válido"},
	"O idioma de exibição só funciona se a ISO do Windows contiver esse idioma. Se a mídia não tiver, a instalação para com erro.":                      {"The display language only works if the Windows ISO contains that language. If the media does not, setup stops with an error.", "El idioma de visualización solo funciona si la ISO de Windows incluye ese idioma. Si el medio no lo tiene, la instalación termina con error."},
	"O script de ajustes (Winhancements.ps1) está desligado: nenhuma remoção de apps ou ajuste de registro será aplicado.":                              {"The tweaks script (Winhancements.ps1) is off: no app removal or registry tweak will be applied.", "El script de ajustes (Winhancements.ps1) está desactivado: no se aplicará ninguna eliminación de aplicaciones ni ajuste del registro."},
	"A senha do usuário fica em TEXTO PURO dentro do XML. Não compartilhe nem publique o arquivo.":                                                      {"The user password is stored as PLAIN TEXT inside the XML. Do not share or publish the file.", "La contraseña del usuario queda en TEXTO PLANO dentro del XML. No comparta ni publique el archivo."},
	"ATENÇÃO: a instalação vai APAGAR o disco %d sem perguntar. Use somente em máquina que pode ser formatada.":                                         {"WARNING: setup will ERASE disk %d without asking. Use only on a machine that can be wiped.", "ATENCIÓN: la instalación BORRARÁ el disco %d sin preguntar. Úsela solo en un equipo que se pueda formatear."},
	"Pós-instalação ligada, mas nenhum programa foi escolhido: o script será criado vazio.":                                                             {"Post-install is on but no program was chosen: the script will be created empty.", "Posinstalación activada, pero no se eligió ningún programa: el script se creará vacío."},
	"ID do winget inválido: %q (use o formato Empresa.Programa)":                                                                                        {"Invalid winget ID: %q (use the format Publisher.Program)", "ID de winget no válido: %q (use el formato Empresa.Programa)"},
	"Os adaptadores de rede serão desativados durante o OOBE sem o BypassNRO; confira se há conta local definida.":                                      {"Network adapters will be disabled during OOBE without BypassNRO; check that a local account is defined.", "Los adaptadores de red se desactivarán durante el OOBE sin BypassNRO; compruebe que haya una cuenta local definida."},
	"O ajuste \"Aplicar ajustes de usuário no primeiro logon\" está desligado: os ajustes de usuário (HKCU) e o script de usuário NÃO serão aplicados.": {"The \"Apply user tweaks at first logon\" item is off: user tweaks (HKCU) and the user script will NOT be applied.", "El ajuste \"Aplicar ajustes de usuario en el primer inicio de sesión\" está desactivado: los ajustes de usuario (HKCU) y el script de usuario NO se aplicarán."},
	"O Microsoft Edge será removido. Tenha outro navegador (instale após o primeiro logon).":                                                            {"Microsoft Edge will be removed. Have another browser ready (install it after first logon).", "Se eliminará Microsoft Edge. Tenga otro navegador (instálelo tras el primer inicio de sesión)."},
	"O UAC (aviso de elevação) será desativado pelo ajuste de privacidade e segurança.":                                                                 {"UAC (elevation prompt) will be turned off by the privacy and security tweak.", "UAC (aviso de elevación) se desactivará con el ajuste de privacidad y seguridad."},
	"O usuário %q aparece mais de uma vez": {"The user %q appears more than once", "El usuario %q aparece más de una vez"},
	"No máximo 20 usuários adicionais":     {"At most 20 additional users", "Como máximo 20 usuarios adicionales"},
	"A senha fica apenas CODIFICADA (Base64) no XML, não criptografada: quem tiver o arquivo consegue ler. Não compartilhe nem publique o arquivo.":                                                                          {"The password is only ENCODED (Base64) in the XML, not encrypted: anyone with the file can read it. Do not share or publish the file.", "La contraseña solo queda CODIFICADA (Base64) en el XML, no cifrada: quien tenga el archivo puede leerla. No comparta ni publique el archivo."},
	"Nenhum usuário local definido. Nas versões novas do Windows 11 (25H2 em diante) a Microsoft removeu os atalhos de conta local (BypassNRO); o jeito garantido é ligar \"Criar um usuário local\" na aba Conta e região.": {"No local user defined. In newer Windows 11 versions (25H2 onward) Microsoft removed the local-account shortcuts (BypassNRO); the reliable way is to turn on \"Create a local user\" in the Account & region tab.", "No hay usuario local definido. En las versiones nuevas de Windows 11 (25H2 en adelante) Microsoft eliminó los atajos de cuenta local (BypassNRO); la forma segura es activar \"Crear un usuario local\" en la pestaña Cuenta y región."},
	"Hyper-V e Área Restrita do Windows só existem nas edições Pro, Education e Enterprise; no Home o comando é ignorado.":                                                                                                   {"Hyper-V and Windows Sandbox only exist in Pro, Education and Enterprise editions; on Home the command is ignored.", "Hyper-V y Espacio aislado de Windows solo existen en las ediciones Pro, Education y Enterprise; en Home el comando se ignora."},
	"O XML gerado ficou inválido (%v). Revise os comandos e scripts próprios.":                                                                                                                                               {"The generated XML is invalid (%v). Review your own commands and scripts.", "El XML generado no es válido (%v). Revise sus comandos y scripts propios."},
}

func tr(lang, pt string) string {
	m, ok := msgs[pt]
	if !ok {
		return pt
	}
	switch lang {
	case "en":
		return m[0]
	case "es":
		return m[1]
	}
	return pt
}
