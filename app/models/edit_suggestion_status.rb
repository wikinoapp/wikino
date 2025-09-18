# typed: strict
# frozen_string_literal: true

class EditSuggestionStatus < T::Enum
  enums do
    Draft = new("draft")
    Open = new("open")
    Applied = new("applied")
    Closed = new("closed")
  end
end
