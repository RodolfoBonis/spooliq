import os
import openai
from github import Github
from github import Auth

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")
REPO_NAME = os.getenv("REPO_NAME")
PR_NUMBER = os.getenv("PR_NUMBER")

openai.api_key = OPENAI_API_KEY

with open('lint_output.txt', 'r') as file:
    lint_output = file.read().strip()

# Se não houver saída de lint, não comenta nada
if not lint_output:
    print("Nenhum problema de lint encontrado. Nenhum comentário será criado.")
    exit(0)

prompt = f"""
Você é um engenheiro de software sênior revisando um pull request. O CI identificou os seguintes problemas de lint e qualidade de código Go. Para cada problema, gere um comentário técnico claro e objetivo, explicando:

* **Descrição do Problema:** Explique o que está errado e por que é importante corrigir.
* **Localização:** Se possível, indique o arquivo/trecho afetado.
* **Sugestão de Correção:** Dê dicas práticas de como resolver, incluindo comandos ou exemplos se necessário.

Os problemas encontrados foram:
{lint_output}

Formate sua resposta como uma lista numerada em markdown, com um item para cada problema identificado.
"""

response = openai.chat.completions.create(
    model="gpt-4o-mini",
    messages=[
        {"role": "system", "content": prompt},
    ],
)

detailed_report = response.choices[0].message.content.strip()

auth = Auth.Token(GITHUB_TOKEN)
git = Github(auth=auth)
repo = git.get_repo(REPO_NAME)
pull_request = repo.get_pull(int(PR_NUMBER))

comment_body = f"### Problemas de Lint encontrados pelo CI\n\n{detailed_report}\n\n**Sugestões:**\n\n- Corrija os problemas apontados para garantir a qualidade e padronização do código.\n- Utilize o script `.config/scripts/lint.sh` localmente para validar antes de subir novas alterações."

pull_request.create_issue_comment(comment_body) 