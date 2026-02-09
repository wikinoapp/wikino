# typed: strict
# frozen_string_literal: true

class TopicMemberRole < T::Enum
  enums do
    Admin = new("admin")
    Member = new("member")
  end
end
