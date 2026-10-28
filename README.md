# Gerador-autounattend - Xerife Tech

Programa para Windows (.exe) que cria o arquivo autounattend.xml, usado para instalar o Windows 10 ou 11 sem precisar clicar nas telas de configuração. Você escolhe as opções numa interface visual e o programa gera o XML pronto. Ele parte de um modelo no estilo Winhance e deixa tudo configurável antes da geração.

<img width="1887" height="1229" alt="image" src="https://github.com/user-attachments/assets/bb7feeaa-eac6-4a49-b8dc-0b9d94ab7bd3" />

Como funciona
Você abre o .exe e uma janela própria aparece, usando o Edge ou o Chrome que já estão no Windows. Se nenhum dos dois for encontrado, abre no navegador padrão.
Não precisa instalar nada, e o programa não exige internet para gerar o arquivo.
Tudo roda localmente, só no seu PC.

Ao terminar, você pode salvar o XML com a janela nativa do Windows, visualizar antes de salvar ou carregar uma configuração salva.
Ao fechar a ferramenta, o canal Xerife Tech no YouTube abre automaticamente.

Recursos principais
Mais de 400 opções configuráveis, divididas em abas (sistema, privacidade, desempenho, interface, remoção de aplicativos e outras).
Presets para começar rápido, e perfis que você pode salvar e reabrir depois.
Conta e instalação: nome de usuário, idioma, região, fuso horário e configurações da instalação.
Avisos de risco: as opções mais arriscadas aparecem sinalizadas, e a ferramenta alerta antes de gerar. Há uma confirmação extra para a opção que apaga o disco.
Aba Pós-instalação: cria na área de trabalho um script que instala programas via winget. Vem com os mais usados já marcados (Chrome, 7-Zip, VLC, Adobe Reader, WhatsApp, AnyDesk, Notepad++ e os runtimes VCRedist e .NET). Há cerca de 50 programas por categoria e um campo para IDs do winget que você quiser acrescentar. Dá para escolher se o script roda sozinho no primeiro login e se se apaga depois de executar.
Segurança de rede: as opções que poderiam afetar internet ou Wi-Fi foram removidas. A opção “Install Winhance” e o atalho dela na área de trabalho também saíram.

Visual e idioma

Interface leve, em azul e branco, no estilo informática.
Modo claro e escuro, à sua escolha.
Três idiomas: português (Brasil), inglês e espanhol. A escolha de idioma e de tema fica salva.
Crédito “Criado por Xerife Tech” no topo e ícone próprio.

Compatibilidade
Windows 10 e Windows 11, em todas as versões, incluindo a 26H2.
Versões para 64 bits, 32 bits (x86) e ARM64.
O XML gerado vale para as instalações que aceitam autounattend.xml.

