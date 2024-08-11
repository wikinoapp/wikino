# typed: strict
# frozen_string_literal: true

class Role < T::Enum
  enums do
    Owner = new("owner")
  end
end
