# Go-orchestrator

Оркестратор, который принимает выражения, сохраняет и отправляет их на агентов. Агентов может быть запущено неограниченное количество: оркестратор зарегистрирует каждый, и любой агент сможет вычислять выражения. После вычисления выражения агент отправляет сообщение обратно на оркестратор, который записывает изменения в бд.

## Как запустить?
Для начала запустите **Docker** и перейдите в терминале в корневую папку проекта
Для запуска сторонних библиотек есть файл **docker-compose.yml**. Его можно запустить следующим образом:
```bash
docker-compose up -d
```
После этой команды в фоновом режиме запустится **RabbitMQ**.
*Важно!**
Для запуска оркестратора нужно написать
```bash
go run orchestrator/main.go
```
Далее для каждого агента потребуется отдельное окно в терминале. Запускается агент из корневой папки проекта командой
```bash
go run agent/main.go
```

## Как работать с проектом?
К сожалению, Frontend проекта я реализовать не успел, поэтому пользоваться оркестратором придется через **Postman**. **Postman** я считаю более наглядным и удобным для тестирования любых API, поэтому, если у вас его нет, советую скачать и проверять через него. Так же он обязательно пригодится в дальнейшей работе Backend-разработчика.
Эндпоинты оркестратора:
### ***http://localhost:8080/expressions*** - При получении *GET* запроса возвращает список всех выражений.

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/5533fbe9-2e4e-443c-a41a-434bee53c5c3)
### ***http://localhost:8080/expressions*** - При получении *POST* запроса создает новое выражение и отправляет его в очередь. **Важно!** Не забудьте указать тело запроса, как в примере.
**Очень важно!** Валидация выражений работает, однако для нее все равно все символы выражения должны быть записаны через пробел, за исключением отрицательных чисел.
(Пример: "1 + 1" <- подходит, "1 + -1" <- подходит, "1+1" <- не подходит)

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/e7f4375c-641e-4935-80fd-ef236d49f897)

### ***http://localhost:8080/agents*** - При получении *GET* запроса возвращает список всех агентов.

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/5d286e1f-fcc4-4f95-a07d-a9491afe6278)

### ***http://localhost:8080/expressions/{id}*** - При получении *GET* запроса возвращает выражение по id

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/3cab6c66-9bff-406d-a3ac-88a0ff51fefc)

### ***http://localhost:8080/timeouts*** - При получении *GET* запроса возвращает время выполнения каждой операции

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/be63146a-6551-4af5-93df-01122f4cf3e2)
