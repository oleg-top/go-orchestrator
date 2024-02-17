package rpn

import (
	"errors"
	"strconv"
	"strings"
)

type RPN struct {
	SNExpression  string
	RPNExpression string
}

func NewRPN(expression string) (*RPN, error) {
	rpn := &RPN{}
	rpn.SNExpression = strings.TrimSpace(expression)
	err := rpn.convertToRPN()
	if err != nil {
		return nil, err
	}
	return rpn, nil
}

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
		if isNumeric(token) || (token == "-" && (i == 0 || !isNumeric(tokens[i-1]))) {
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

func isOperator(char rune) bool {
	return char == '+' || char == '-' || char == '*' || char == '/'
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// func main() {
//	expression := "-3 - 1 * 3 * 123*"
//	rpn, err := NewRPN(expression)
//	if err != nil {
//		fmt.Println(err.Error())
//	} else {
//		fmt.Println(rpn.RPNExpression)
//	}
//}
