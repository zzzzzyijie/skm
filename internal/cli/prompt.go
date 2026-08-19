package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
)

type promptView struct {
	domain.Prompt
	Content string `json:"content"`
	Body    string `json:"body"`
}

func (a *App) newPromptCommand() *cobra.Command {
	command := &cobra.Command{Use: "prompt", Short: "Manage reusable Prompt templates", Args: cobra.NoArgs}
	command.AddCommand(
		a.newPromptCreateCommand(),
		a.newPromptAddCommand(),
		a.newPromptListCommand(),
		a.newPromptShowCommand(),
		a.newPromptValidateCommand(),
		a.newPromptUpdateCommand(),
		a.newPromptRenderCommand(),
		a.newPromptExportCommand(),
		a.newPromptRemoveCommand(),
	)
	return command
}

func (a *App) newPromptCreateCommand() *cobra.Command {
	var description, body, bodyFile string
	var tagValues, variableDefinitions []string
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a local Prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(description) == "" {
				return fmt.Errorf("description is required")
			}
			if body != "" && bodyFile != "" {
				return fmt.Errorf("use only one of --body or --body-file")
			}
			if bodyFile != "" {
				data, err := os.ReadFile(bodyFile)
				if err != nil {
					return err
				}
				body = string(data)
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("body is required")
			}
			variables, err := promptCreateVariables(variableDefinitions)
			if err != nil {
				return err
			}
			content, err := promptpkg.Build(args[0], description, body, tagValues, variables)
			if err != nil {
				return err
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Prompt
			err = withLock(storage, func() error {
				value, err = promptpkg.New(storage).Create(string(content), "local", tagValues)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("prompt create", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Created Prompt %s\n", value.ID)
				return err
			})
		},
	}
	command.Flags().StringVar(&description, "description", "", "Prompt description")
	command.Flags().StringVar(&body, "body", "", "Prompt template body")
	command.Flags().StringVar(&bodyFile, "body-file", "", "read Prompt template body from a file")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "Prompt tag (repeatable)")
	command.Flags().StringArrayVar(&variableDefinitions, "variable", nil, "required variable as name or name:type (repeatable)")
	return command
}

func (a *App) newPromptAddCommand() *cobra.Command {
	var sourceName string
	var tagValues []string
	command := &cobra.Command{
		Use:   "add <prompt-path>",
		Short: "Add a PROMPT.md file to the Prompt Library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Prompt
			err = withLock(storage, func() error {
				value, err = promptpkg.New(storage).Add(args[0], sourceName, tagValues)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("prompt add", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Added %s to Prompt Library; tags=%s\n", value.ID, strings.Join(value.Tags, ","))
				return err
			})
		},
	}
	command.Flags().StringVar(&sourceName, "source", "", "local Prompt source name")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "Prompt tag (repeatable)")
	return command
}

func (a *App) newPromptListCommand() *cobra.Command {
	var tagValues []string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Prompts",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			values, err := promptpkg.New(storage).List(tagValues)
			if err != nil {
				return err
			}
			return a.emit("prompt list", values, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "ID\tTAGS\tVARIABLES\tDESCRIPTION")
				for _, value := range values {
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%d\t%s\n", value.ID, strings.Join(value.Tags, ","), len(value.Variables), singleLine(value.Description))
				}
				return writer.Flush()
			})
		},
	}
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "require a Prompt tag (repeatable, AND semantics)")
	return command
}

func (a *App) newPromptShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <prompt>",
		Short: "Show Prompt metadata and template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			value, document, err := promptpkg.New(storage).Read(args[0])
			if err != nil {
				return err
			}
			view := promptView{Prompt: value, Content: document.Content, Body: document.Body}
			return a.emit("prompt show", view, func() error {
				_, err := fmt.Fprintf(a.Out, "ID: %s\nDescription: %s\nTags: %s\nVariables: %d\nHash: %s\n\n%s",
					value.ID, value.Description, strings.Join(value.Tags, ", "), len(value.Variables), value.Hash, document.Content)
				return err
			})
		},
	}
}

func (a *App) newPromptValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <prompt-path>",
		Short: "Validate a PROMPT.md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := promptpkg.Validate(args[0])
			if err != nil {
				return err
			}
			document.Content = ""
			document.Body = ""
			return a.emit("prompt validate", document, func() error {
				_, err := fmt.Fprintf(a.Out, "Valid Prompt: %s (%d variables)\n", document.Name, len(document.Variables))
				return err
			})
		},
	}
}

func (a *App) newPromptUpdateCommand() *cobra.Command {
	var baseHash string
	var tagValues []string
	command := &cobra.Command{
		Use:   "update <prompt> <prompt-path>",
		Short: "Update a Prompt from a PROMPT.md file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := promptpkg.Validate(args[1])
			if err != nil {
				return err
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Prompt
			err = withLock(storage, func() error {
				value, err = promptpkg.New(storage).Update(args[0], document.Content, baseHash, tagValues)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("prompt update", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Updated Prompt %s\n", value.ID)
				return err
			})
		},
	}
	command.Flags().StringVar(&baseHash, "base-hash", "", "reject the update if the stored Prompt hash changed")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "replace Prompt tags (repeatable)")
	return command
}

func (a *App) newPromptRenderCommand() *cobra.Command {
	var variableValues, variableFiles []string
	command := &cobra.Command{
		Use:   "render <prompt>",
		Short: "Render a Prompt with variable values",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			_, document, err := promptpkg.New(storage).Read(args[0])
			if err != nil {
				return err
			}
			values, err := promptVariables(variableValues, variableFiles)
			if err != nil {
				return err
			}
			result, err := promptpkg.Render(document, values)
			if err != nil {
				return err
			}
			if len(result.MissingVariables) > 0 {
				return fmt.Errorf("missing required prompt variables: %s", strings.Join(result.MissingVariables, ", "))
			}
			return a.emit("prompt render", result, func() error {
				_, err := fmt.Fprint(a.Out, result.Content)
				return err
			})
		},
	}
	command.Flags().StringArrayVar(&variableValues, "var", nil, "Prompt variable as name=value (repeatable)")
	command.Flags().StringArrayVar(&variableFiles, "var-file", nil, "Prompt variable read from a file as name=path (repeatable)")
	return command
}

func (a *App) newPromptExportCommand() *cobra.Command {
	var output string
	command := &cobra.Command{
		Use:   "export <prompt>",
		Short: "Export a Prompt as PROMPT.md",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			value, document, err := promptpkg.New(storage).Read(args[0])
			if err != nil {
				return err
			}
			if output == "" {
				return a.emit("prompt export", map[string]any{"prompt": value, "content": document.Content}, func() error {
					_, err := fmt.Fprint(a.Out, document.Content)
					return err
				})
			}
			if err := fsx.AtomicWriteFile(output, []byte(document.Content), 0o644); err != nil {
				return err
			}
			return a.emit("prompt export", map[string]any{"prompt": value, "output": output}, func() error {
				_, err := fmt.Fprintf(a.Out, "Exported %s to %s\n", value.ID, output)
				return err
			})
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "", "write PROMPT.md to this path instead of stdout")
	return command
}

func (a *App) newPromptRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <prompt>",
		Short: "Remove a Prompt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Prompt
			err = withLock(storage, func() error {
				value, err = promptpkg.New(storage).Remove(args[0])
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("prompt remove", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Removed Prompt %s\n", value.ID)
				return err
			})
		},
	}
}

func promptVariables(values, files []string) (map[string]string, error) {
	result := make(map[string]string, len(values)+len(files))
	for _, assignment := range values {
		name, value, err := splitPromptAssignment(assignment)
		if err != nil {
			return nil, err
		}
		result[name] = value
	}
	for _, assignment := range files {
		name, path, err := splitPromptAssignment(assignment)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read prompt variable %s: %w", name, err)
		}
		if int64(len(data)) > promptpkg.MaxPromptSize {
			return nil, fmt.Errorf("prompt variable %s exceeds %d bytes", name, promptpkg.MaxPromptSize)
		}
		result[name] = string(data)
	}
	return result, nil
}

func splitPromptAssignment(value string) (string, string, error) {
	name, content, ok := strings.Cut(value, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return "", "", fmt.Errorf("Prompt variable must use name=value")
	}
	return name, content, nil
}

func promptCreateVariables(definitions []string) ([]domain.PromptVariable, error) {
	result := make([]domain.PromptVariable, 0, len(definitions))
	seen := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		name, variableType, hasType := strings.Cut(strings.TrimSpace(definition), ":")
		if name == "" || (hasType && strings.TrimSpace(variableType) == "") {
			return nil, fmt.Errorf("Prompt variable must use name or name:type")
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate Prompt variable %q", name)
		}
		seen[name] = true
		if !hasType {
			variableType = "text"
		}
		variableType = strings.TrimSpace(variableType)
		if variableType == "select" {
			return nil, fmt.Errorf("select variables require options; create or edit a PROMPT.md file instead")
		}
		result = append(result, domain.PromptVariable{Name: name, Label: name, Type: variableType, Required: true})
	}
	return result, nil
}
