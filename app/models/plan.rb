# typed: strict
# frozen_string_literal: true

class Plan < T::Enum
  enums do
    Free = new("free")
    Small = new("small")
    Large = new("large")
  end
end
