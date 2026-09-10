# Карта стандартов Agent-Ops
## Информативное сопоставление v0.4.0

**Исходная ревизия aom-03-r12 | приоритет английской версии**

> Этот документ является информативной картой; названные внешние источники сохраняют приоритет для собственных требований.

## 1. Интерпретация

Сопоставление означает, что контроль Agent-Ops может дать доказательство, относящееся к внешней задаче. Результат такого сопоставления НЕ ДОЛЖЕН читаться как сертификация, юридическая консультация или доказательство полного соответствия. Каждое заявление подлежит проверке на конкретном внедрении.

Сопоставление с ISO/IEC 42001 выполнено Git in Sky на основе карты «риск — контроль», опубликованной в OWASP Agentic Skills Top 10. OWASP такого сопоставления для Agent-Ops не выполняла и его не подтверждала.

## 2. Подробная карта контролей

| Раздел Agent-Ops | ISO/IEC 42001, Приложение A | Внешний каталог рисков |
| --- | --- | --- |
| §7 Evidence (доказательства), §16.5 запись покрытия проверки агентного артефакта | A.6.2.4 Верификация и валидация | AST08 Poor Scanning; AISVS C11 (состязательное тестирование) |
| §15 заявленное и наблюдаемое состояние, метаданные агентных артефактов | A.6.2.7 Техническая документация | AST04 Insecure Metadata |
| §16 Project Operations Harness (проектная эксплуатационная обвязка), §16.2 машиночитаемые разрешения | A.6.2.5 Развёртывание | AST03 Over-Privileged Skills; AISVS C5, C9 |
| §16.1 иерархия доверия, §20.1 безопасность, §20.3 жизненный цикл контекста | A.6.2.6 Эксплуатация и мониторинг | AST05 Untrusted External Instructions; AST07 Update Drift |
| §16.5 инвентарь агентных артефактов | A.10.3 Поставщики | AST02 Supply Chain Compromise |
| §18 Governance Mesh (сквозной слой управляющих правил), §18.1 квитанции допуска и исхода | A.2.2 Политика в области ИИ | AST09 No Governance; Регламент (ЕС) 2024/1689, статья 12 |
| §9 Plan (план), §11 Controlled Change (контролируемое изменение) и Gated Executor (исполнитель через контрольные ворота), §20.1 разделение учётных данных | A.4.5 Системы и вычислительные ресурсы | AST06 Weak Isolation |
| §20.1 прикладные контроли безопасности и инвариант запрета изменений | A.6.2.4 Верификация и валидация | AST01 Malicious Skills; AST10 Cross-Platform Reuse |

## 3. Стандарты и нормативные акты

- ISO/IEC 42001:2023, Information technology - Artificial intelligence - Management system (система менеджмента искусственного интеллекта): <https://www.iso.org/standard/42001>
- Регламент (ЕС) 2024/1689 (EU AI Act), статья 12 «Ведение журналов» и статья 113 «Сроки применения»: <https://eur-lex.europa.eu/eli/reg/2024/1689/oj/eng>
- NIST AI Risk Management Framework (рамочная модель управления рисками ИИ), функция GOVERN: <https://www.nist.gov/itl/ai-risk-management-framework>
- SLSA, модель происхождения и целостности сборки: <https://slsa.dev/>

## 4. Каталоги рисков и стандарты проверки

- OWASP Agentic Skills Top 10 (AST01-AST10), версия 1.0, 2026: <https://owasp.org/www-project-agentic-skills-top-10/>; репозиторий: <https://github.com/OWASP/www-project-agentic-skills-top-10>
- OWASP AI Security Verification Standard (AISVS) — стандарт проверки того, реализован ли контроль: <https://github.com/OWASP/AISVS>
- OWASP Top 10 for Agentic Applications (инициатива Agentic Security Initiative, ASI): <https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/>
- OWASP MCP Top 10 — риски серверов Model Context Protocol, смежная поверхность, не покрываемая AST: <https://owasp.org/www-project-mcp-top-10/>
- OWASP Gen AI Security Project, включая Top 10 for LLM Applications: <https://genai.owasp.org/>
- CSA MAESTRO — семислойная модель угроз агентных систем: <https://cloudsecurityalliance.org/blog/2025/02/06/agentic-ai-threat-modeling-framework-maestro>; репозиторий: <https://github.com/CloudSecurityAlliance/MAESTRO>

## 5. Проанализированные системы и материалы по практическим случаям безопасности

Следующие системы и публикации использовались при сравнительном анализе архитектуры и модели угроз. Их включение носит информативный характер: оно не является рекомендацией использовать продукт, нормативной зависимостью или доказательством соответствия названной системы методологии Agent-Ops.

- HolmesGPT — открытый SRE-агент (агент для инженерии надёжности сервисов) для расследования инцидентов в рабочей инфраструктуре: репозиторий <https://github.com/HolmesGPT/holmesgpt>; документация <https://holmesgpt.dev/latest/>.
- Warden — шлюз с учётом идентичности, опосредующий доступ агента к внешним системам: репозиторий <https://github.com/stephnangue/warden>; документация <https://wardengateway.com/>.
- Hugging Face, «Anatomy of a Frontier Lab Agent Intrusion: A Technical Timeline of the July 2026 Incident» («Анатомия проникновения агента в инфраструктуру передовой лаборатории: техническая хронология инцидента июля 2026 года»): <https://huggingface.co/blog/agent-intrusion-technical-timeline>.
- Docker, «A new security baseline for enterprise agentic adoption» («Новый базовый уровень безопасности для корпоративного внедрения агентных систем»): <https://www.docker.com/blog/a-new-security-baseline-for-enterprise-agentic-adoption/>.
- Cloudflare, «The Agent Access Model» («Модель доступа агентов»): <https://blog.cloudflare.com/the-agent-access-model/>.
- Ganesh Gurudu, «Building an AI Agent That Runs Your SRE Operations - What I Learned, What Works, and How You Can Do It Too» («Создание ИИ-агента, который ведёт SRE-эксплуатацию: извлечённые уроки, рабочие подходы и способ повторить»), 8 апреля 2026 года: <https://blog.stackademic.com/building-an-ai-agent-that-runs-your-sre-operations-what-i-learned-what-works-and-how-you-can-do-8a3801124bdc>. Статья используется как информативный источник опыта реализации; Agent-Ops не перенимает из неё набор продуктов, шкалу риска, топологию размещения или заявления о точности.
- Ali Shazal и Matthew Koen, Anthropic, «A guide to the anatomy of effective commerce agents» («Руководство по устройству эффективных торговых агентов»), 2 сентября 2026 года: <https://claude.com/blog/the-anatomy-of-effective-commerce-agents>. Статья используется как информативный источник за пределами эксплуатационного домена; Agent-Ops не перенимает из неё торговую архитектуру, выбор модели, приёмы снижения задержки или стек развёртывания.

## 6. Оговорки

- OWASP AIVSS (AI Vulnerability Scoring System — система оценки серьёзности уязвимости ИИ, <https://aivss.owasp.org/>) в Agent-Ops не применяется: на момент этой редакции версия 1.0 не выпущена, и сам OWASP Agentic Skills Top 10 воздерживается от присвоения оценок серьёзности до её выхода.
- Фактические наблюдения об инцидентах, кампаниях и результатах обхода сканеров, на которые опирается OWASP Agentic Skills Top 10, в настоящем документе не воспроизводятся и независимо не проверялись. Ссылки на первоисточники приведены в самом OWASP AST и в отдельном разборе применимости, публикуемом вместе с методологией.
- Соответствие внешнему каталогу рисков означает, что методология адресует названный риск, а не что риск устранён в конкретном внедрении. Устранение подтверждается доказательствами прогона, а не текстом.

## 7. Использование доказательств

Реализациям СЛЕДУЕТ фиксировать точную редакцию внешнего источника, применимый пункт, локальный контроль и идентичность доказательства. Общего значка стандарта недостаточно.

## 8. Управление изменениями

Каждый внешний источник МОЖЕТ меняться независимо. Обновление карты создаёт новую исходную ревизию и не переписывает прежние доказательства.

## 9. Юридическая граница

Ответственный орган ДОЛЖЕН определить юридическую применимость и сохранить своё решение. Agent-Ops предоставляет доказательства жизненного цикла, а не юридическое полномочие.
