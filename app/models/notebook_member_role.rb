# typed: strict
# frozen_string_literal: true

class NotebookMemberRole < T::Enum
  enums do
    Admin = new("admin")
    Member = new("member")
  end
end
