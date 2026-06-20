package selo

import "strings"

// personFirstNames is an expanded pool of common pt-BR given names (balanced
// male/female, classic + modern). Sampled uniformly by GeneratePerson. Accents
// are limited to the set covered by asciiFold so email local-parts stay ASCII.
var personFirstNames = []string{
	"João", "Maria", "José", "Ana", "Carlos", "Beatriz", "Pedro", "Juliana",
	"Lucas", "Fernanda", "Rafael", "Camila", "Bruno", "Larissa", "Gabriel", "Mariana",
	"Miguel", "Arthur", "Heitor", "Helena", "Alice", "Laura", "Davi", "Valentina",
	"Sophia", "Enzo", "Lorenzo", "Manuela", "Cecília", "Bernardo", "Matheus", "Letícia",
	"Guilherme", "Gustavo", "Vinícius", "Eduardo", "Henrique", "Felipe", "Daniel", "Leonardo",
	"Caio", "Diego", "André", "Marcelo", "Ricardo", "Rodrigo", "Fábio", "Marcos",
	"Paulo", "Roberto", "Antônio", "Francisco", "Sérgio", "Júlio", "Fernando", "Luís",
	"Renato", "Otávio", "Murilo", "Samuel", "Benício", "Anthony", "Theo", "Pietro",
	"Isaac", "Nicolas", "Bryan", "Yuri", "Igor", "Thiago", "Alexandre", "Wesley",
	"Adriana", "Patrícia", "Aline", "Bruna", "Carolina", "Débora", "Elaine", "Gabriela",
	"Isabela", "Jéssica", "Kelly", "Lívia", "Marcela", "Natália", "Priscila", "Renata",
	"Sabrina", "Tatiane", "Vanessa", "Yasmin", "Amanda", "Bianca", "Carla", "Daniela",
	"Eduarda", "Flávia", "Giovana", "Heloísa", "Isadora", "Júlia", "Lara", "Luiza",
	"Melissa", "Nicole", "Pâmela", "Rafaela", "Sara", "Tainá", "Vitória", "Yara",
	"Clara", "Sofia", "Antônia", "Cristiane", "Rosângela", "Mônica", "Verônica", "Lúcia",
}

// personSurnames is an expanded pool of common pt-BR family names. Sampled by
// GeneratePerson and reused as street-name tokens by genAddressForUFRand, so a
// richer list also diversifies synthesized logradouros. Accents are limited to
// the set covered by asciiFold.
var personSurnames = []string{
	"Silva", "Santos", "Oliveira", "Souza", "Lima", "Pereira", "Ferreira",
	"Costa", "Rodrigues", "Almeida", "Nascimento", "Carvalho",
	"Gomes", "Martins", "Araújo", "Ribeiro", "Barbosa", "Rocha", "Dias", "Teixeira",
	"Cardoso", "Moreira", "Cavalcanti", "Nogueira", "Pinto", "Moraes", "Mendes", "Freitas",
	"Barros", "Vieira", "Ramos", "Monteiro", "Castro", "Campos", "Cunha", "Correia",
	"Andrade", "Fernandes", "Lopes", "Marques", "Machado", "Azevedo", "Melo", "Reis",
	"Tavares", "Borges", "Farias", "Coelho", "Pinheiro", "Cruz", "Duarte", "Sampaio",
	"Aragão", "Bezerra", "Brito", "Caldeira", "Camargo", "Caetano", "Esteves", "Fonseca",
	"Garcia", "Guimarães", "Henriques", "Leite", "Macedo", "Magalhães", "Maia", "Medeiros",
	"Neves", "Pacheco", "Peixoto", "Queiroz", "Sales", "Santana", "Siqueira", "Viana",
	"Xavier", "Antunes", "Bastos", "Brandão", "Furtado", "Galvão",
}

// personCompanySuffixes are the legal-form tokens appended to a synthetic
// company name.
var personCompanySuffixes = []string{"ME", "LTDA", "S.A.", "EIRELI", "Comércio", "Serviços"}

// asciiFold replaces the accented characters used in the name lists with their
// ASCII equivalents, for building email local-parts.
var asciiFold = strings.NewReplacer(
	"ã", "a", "á", "a", "â", "a", "à", "a",
	"é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
)
