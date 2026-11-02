package core

// BusinessService encapsulates core business logic and validations,
// separating domain/business rules from Fyne UI event handlers.
type BusinessService struct {
	state *AppState
}

// NewBusinessService creates a new BusinessService instance.
func NewBusinessService(state *AppState) *BusinessService {
	return &BusinessService{state: state}
}

// HasQuestions checks if the current state has questions loaded.
func (s *BusinessService) HasQuestions() bool {
	return s.state != nil && len(s.state.Questions) > 0
}

// HasWrongSet checks if the current state has wrong questions.
func (s *BusinessService) HasWrongSet() bool {
	return s.state != nil && len(s.state.WrongSet) > 0
}

// HasFavSet checks if the current state has favorite questions.
func (s *BusinessService) HasFavSet() bool {
	return s.state != nil && len(s.state.FavSet) > 0
}

// IsBankSelected checks if a current storage bank key is selected.
func (s *BusinessService) IsBankSelected() bool {
	return s.state != nil && s.state.CurrentStorageKey != ""
}
