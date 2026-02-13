# typed: strict
# frozen_string_literal: true

class SpaceMemberRole < T::Enum
  enums do
    Owner = new("owner")
    Member = new("member")
  end
end
