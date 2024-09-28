# typed: strict
# frozen_string_literal: true

class NotebookVisibility < T::Enum
  enums do
    Public = new("public")
    Private = new("private")
  end
end
