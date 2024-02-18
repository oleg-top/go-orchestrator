# Go-orchestrator

Оркестратор, который принимает выражения, сохраняет и отправляет их на агентов. Агентов может быть запущено неограниченное количество: оркестратор зарегистрирует каждый, и любой агент сможет вычислять выражения. После вычисления выражения агент отправляет сообщение обратно на оркестратор, который записывает изменения в бд. (Подробнее о работе оркестратора ниже)

## Как запустить?
Для начала запустите **Docker** и перейдите в терминале в корневую папку проекта
Для запуска сторонних библиотек есть файл **docker-compose.yml**. Его можно запустить следующим образом:
```bash
docker-compose up -d
```
После этой команды в фоновом режиме запустится **RabbitMQ**.
*Важно!*
Не забудьте после использования проекта выключить docker-compose. Это можно сделать следующей командой:
```bash
docker-compose down
```
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
### ***http://localhost:8080/expressions*** - При получении *POST* запроса создает новое выражение и отправляет его в очередь. *Важно!* Не забудьте указать тело запроса, как в примере.
*Очень важно!* Валидация выражений работает, однако для нее все равно все символы выражения должны быть записаны через пробел, за исключением отрицательных чисел.
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

### ***http://localhost:8080/timeouts*** - При получении *POST* запроса меняет задержки каждой операции. *Важно!* Не забудьте указать тело запроса, как в примере (время каждой операции задается в миллисекундах).

**Пример**:
![image](https://github.com/oleg-top/go-orchestrator/assets/68245949/21683d81-813f-4d05-8914-ef0cc2d762d2)

## Что-либо не работает?
Можете написать мне в тг: *@komarik_sasha*

## Как устроен проект?
### Оркестратор
Запускает сервер, мониторит агентов. Если агент не присылает хартбит пинги в течение тридцати секунд, то он объявляется нерабочим, а выражение, которое он считал, отправляется снова в очередь. Все выражения и агенты хранятся в бд. Для работы с базой данных сделал отдельный package storage.
### Хранилище
Сделал как отдельную структуру для удобной работы с бд. В ней реализовал методы получения информации из бд, ее обновления и тд.
### Агент
Агент следит за очередью и получает, если свободен, новое выражение. Также агент в отдельной горутине постоянно посылает хартбит пинги оркестратору. Получив выражение, агент переводит его обратную польскую нотацию (для этого написал package rpn), проходится по нему, пока выражение не превратится в одно число, при этом запуская горутины для вычисления выражений в один знак. Тем самым обеспечивается параллельность вычислений (например, "2 * 3 + 4 * 3" - параллельно посчитаются "2 * 3" и "4 * 3", потом проссумируются результаты выражений).
### Обратная польская нотация
Сделал как отдельную структуру для удобной работы с обратной польской нотацией. Структура представляет из себя два поля: выражение в стандартной нотации и выражение в обратной польской нотации. В саму польскую нотацию я перевожу засчет весьма нетривиального алгоритма с использованием стеков.
### Сереализация
В RabbitMQ можно передовать только массивы байтов, поэтому я сделал package serialization, для сереализации и десериализации структур сообщений. При помощи интерфейса и дженериков я избавился от лишнего дублирования вышеназванных функций
