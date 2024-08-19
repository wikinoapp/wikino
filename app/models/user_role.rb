# typed: strict
# frozen_string_literal: true

class UserRole < T::Enum
  enums do
    Owner = new("owner")
  end
end
