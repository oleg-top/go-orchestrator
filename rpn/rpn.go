package rpn

import (
	"errors"
	"strconv"
	"strings"
)

// Структура, хранящая в себе выражение в обычной и обратной польской нотациях
type RPN struct {
	SNExpression  string
	RPNExpression string
}

// Создает новый экземпляр структуры RPN
func NewRPN(expression string) (*RPN, error) {
	rpn := &RPN{}
	rpn.SNExpression = strings.TrimSpace(expression)
	err := rpn.convertToRPN()
	if err != nil {
		return nil, err
	}
	return rpn, nil
}

// Проверяет выражение на валидность (все равно нужно все символы выражения писать через пробел, кроме отрицательных чисел)
func (r *RPN) validateExpression() error {
	nonValid := []string{
		"- - ", "- +", "- *", "- /",
		"+ - ", "+ +", "+ *", "+ /",
		"* - ", "* +", "* *", "* /",
		"/ - ", "/ +", "/ *", "/ /",
	}
	for _, el := range nonValid {
		if strings.Contains(r.SNExpression, el) {
			return errors.New("Неправильное расположение знаков")
		}
	}
	if strings.ContainsAny(r.SNExpression, "()") {
		return errors.New("Работа со скобками пока не поддерживается")
	}
	nonOutside := []string{
		"-", "+", "*", "/",
	}
	for _, el := range nonOutside {
		if string(r.SNExpression[0]) == string(el) {
			if el == "-" {
				if string(r.SNExpression[1]) == " " {
					return errors.New("Неправильное расположение знаков")
				}
			} else {
				return errors.New("Неправильное расположение знаков")
			}
		}
		if string(r.SNExpression[len(r.SNExpression)-1]) == string(el) {
			return errors.New("Неправильное расположение знаков")
		}
	}
	return nil
}

// Переводит выражение из обычной в обратную польскую нотацию
func (r *RPN) convertToRPN() error {
	err := r.validateExpression()
	if err != nil {
		return err
	}

	operators := map[string]int{"+": 1, "-": 1, "*": 2, "/": 2}
	var output []string
	var stack []string

	tokens := strings.Fields(r.SNExpression)

	for i, token := range tokens {
		if IsNumeric(token) || (token == "-" && (i == 0 || !IsNumeric(tokens[i-1]))) {
			output = append(output, token)
		} else if token == "(" {
			stack = append(stack, token)
		} else if token == ")" {
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = stack[:len(stack)-1]
		} else {
			for len(stack) > 0 && operators[stack[len(stack)-1]] >= operators[token] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		}
	}

	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	r.RPNExpression = strings.Join(output, " ")
	return nil
}

// Проверяет, является ли символ оператором
func IsOperator(char rune) bool {
	return char == '+' || char == '-' || char == '*' || char == '/'
}

// Проверяет, является ли символ числом
func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// Использовал для тестирования структуры
// func main() {
//	expression := "1 - 3 * 2"
//	rpn, err := NewRPN(expression)
//	if err != nil {
//		fmt.Println(err.Error())
//	} else {
//		fmt.Println(rpn.RPNExpression)
//	}
//}
