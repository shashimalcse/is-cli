package interactive

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shashimalcse/is-cli/internal/core"
	"github.com/shashimalcse/is-cli/internal/management"
	"github.com/shashimalcse/is-cli/internal/tui"
)

type ApplicationCreateState int

const (
	StateInitiated ApplicationCreateState = iota
	StateTypeSelected
	StateQuestionsCompleted
	StateCreatingInProgress
	StateCreatingCompleted
	StateCreatingError
)

type ApplicationType string

const (
	SinglePage  ApplicationType = "Single-Page Application"
	Traditional ApplicationType = "Traditional Web Application"
	Mobile      ApplicationType = "Mobile Application"
	Standard    ApplicationType = "Standard-Based Application"
	M2M         ApplicationType = "M2M Application"
)

type ApplicationCreateModel struct {
	styles               *tui.Styles
	spinner              spinner.Model
	width, height        int
	cli                  *core.CLI
	state                ApplicationCreateState
	stateError           error
	applicationTypes     list.Model
	questions            []tui.Question
	currentQuestionIndex int
	applicationType      ApplicationType
	output               string
}

func NewApplicationCreateModel(cli *core.CLI) *ApplicationCreateModel {
	return &ApplicationCreateModel{
		styles:           tui.DefaultStyles(),
		spinner:          newSpinner(),
		cli:              cli,
		state:            StateInitiated,
		applicationTypes: newApplicationTypesList(),
		questions:        initQuestions(),
	}
}

func newApplicationTypesList() list.Model {
	items := []list.Item{
		tui.NewItemWithKey("single_page", string(SinglePage), "A web application that runs application logic in the browser."),
		tui.NewItemWithKey("traditional", string(Traditional), "A web application that runs application logic on the server."),
		tui.NewItemWithKey("mobile", string(Mobile), "Applications developed to target mobile devices."),
		tui.NewItemWithKey("standard", string(Standard), "Applications built using standard protocols."),
		tui.NewItemWithKey("m2m", string(M2M), "Applications tailored for Machine to Machine communication."),
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select application template to create application"
	return l
}

func initQuestions() []tui.Question {
	return []tui.Question{
		tui.NewQuestion("Name", "Name", tui.ShortQuestion),
		tui.NewQuestion("Authorized redirect URL", "Authorized redirect URL", tui.ShortQuestion),
		tui.NewQuestion("Are you sure you want to create the application? (y/n)", "Are you sure you want to create the application? (Y/n)", tui.ShortQuestion),
	}
}

func (m ApplicationCreateModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ApplicationCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleKeyEnter(msg)
		}
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)
	}

	var cmd tea.Cmd
	if m.state == StateInitiated {
		m.applicationTypes, _ = m.applicationTypes.Update(msg)
	}
	if m.state == StateTypeSelected || m.state == StateQuestionsCompleted {
		m.questions[m.currentQuestionIndex].Input, _ = m.questions[m.currentQuestionIndex].Input.Update(msg)
	}
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m ApplicationCreateModel) handleKeyEnter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateInitiated:
		i, ok := m.applicationTypes.SelectedItem().(tui.Item)
		if ok {
			m.applicationType = ApplicationType(i.Title())
			m.state = StateTypeSelected
		}
	case StateTypeSelected:
		switch m.applicationType {
		case SinglePage:
			currentQuestion := &m.questions[m.currentQuestionIndex]
			currentQuestion.Answer = currentQuestion.Input.Value()
			if m.currentQuestionIndex == len(m.questions)-2 {
				m.state = StateQuestionsCompleted
				m.NextQuestion()
				m.questions[m.currentQuestionIndex].Input.SetValue("")
			} else {
				m.NextQuestion()
			}
			return m, currentQuestion.Input.Blur
		}
	case StateQuestionsCompleted:
		confirmation := strings.ToLower(m.questions[m.currentQuestionIndex].Input.Value())
		if (confirmation == "y") || (confirmation == "Y" || confirmation == "") {
			m.state = StateCreatingInProgress
			err := m.createApplications()
			if err != nil {
				m.state = StateCreatingError
				m.stateError = err
				m.output = "Error creating application!"
			} else {
				m.state = StateCreatingCompleted
				m.output = "Application created successfully!"
			}
		} else {
			m.output = "Application creation cancelled."
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ApplicationCreateModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width, m.height = msg.Width, msg.Height
	h, v := m.styles.List.GetFrameSize()
	m.applicationTypes.SetSize(msg.Width-h, msg.Height-v)
	return m, nil
}

func (m ApplicationCreateModel) View() string {
	switch m.state {
	case StateInitiated:
		return m.styles.List.Render(m.applicationTypes.View())
	case StateTypeSelected, StateQuestionsCompleted:
		return m.renderQuestions()
	case StateCreatingInProgress:
		return fmt.Sprintf("\n\n   %s Creating application...\n\n", m.spinner.View())
	case StateCreatingCompleted:
		return "Application created successfully!"
	case StateCreatingError:
		return fmt.Sprintf("Error creating application: %v", m.stateError)
	}
	return ""
}

func (m *ApplicationCreateModel) renderQuestions() string {
	if m.applicationType != SinglePage {
		return "Not supported yet!"
	}
	var sb strings.Builder
	for i, q := range m.questions[:m.currentQuestionIndex] {
		sb.WriteString(fmt.Sprintf("%s: %s\n", q.Question, q.Answer))
		if i == len(m.questions)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString(m.questions[m.currentQuestionIndex].Input.View())
	return sb.String()
}

func (m ApplicationCreateModel) Value() string {
	return fmt.Sprint(m.output)
}

func (m *ApplicationCreateModel) NextQuestion() {
	if m.currentQuestionIndex < len(m.questions)-1 {
		m.currentQuestionIndex++
	} else {
		m.currentQuestionIndex = 0
	}
}

func (m ApplicationCreateModel) createApplications() error {

	if m.applicationType == SinglePage {
		application := management.Application{
			Name:       m.questions[0].Answer,
			TemplateID: "6a90e4b0-fbff-42d7-bfde-1efd98f07cd7",
			AdvancedConfig: management.AdvancedConfigurations{
				DiscoverableByEndUsers: false,
				SkipLoginConsent:       true,
				SkipLogoutConsent:      true,
			},
			AssociatedRoles: management.AssociatedRoles{
				AllowedAudience: "APPLICATION",
				Roles:           []management.AssociatedRole{},
			},
			AuthenticationSeq: management.AuthenticationSequence{
				Type: "DEFAULT",
				Steps: []management.Step{{
					ID: 1,
					Options: []management.Options{
						{IDP: "LOCAL", Authenticator: "basic"},
					},
				},
				},
			},
			ClaimConfiguration: management.ClaimConfiguration{
				Dialect: "LOCAL",
				RequestedClaims: []interface{}{
					map[string]interface{}{
						"claim": map[string]interface{}{"uri": "http://wso2.org/claims/username"},
					},
				},
			},
			InboundProtocolConfiguration: management.InboundProtocolConfiguration{
				OIDC: management.OIDC{
					AccessToken: management.AccessToken{
						ApplicationAccessTokenExpiryInSeconds: 3600,
						BindingType:                           "sso-session",
						RevokeTokensWhenIDPSessionTerminated:  true,
						Type:                                  "Default",
						UserAccessTokenExpiryInSeconds:        3600,
						ValidateTokenBinding:                  false,
					},
					AllowedOrigins: []string{m.questions[1].Answer},
					CallbackURLs:   []string{m.questions[1].Answer},
					GrantTypes:     []string{"authorization_code", "refresh_token"},
					PKCE: management.PKCE{
						Mandatory:                      true,
						SupportPlainTransformAlgorithm: false,
					},
					PublicClient: true,
					RefreshToken: management.RefreshToken{
						ExpiryInSeconds:   86400,
						RenewRefreshToken: true,
					},
				},
			},
		}
		_, err := m.cli.API.Application.Create(context.Background(), application)
		return err
	}
	return nil
}