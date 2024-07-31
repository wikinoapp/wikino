# typed: strict
# frozen_string_literal: true

class EmailConfirmationEvent < T::Enum
  enums do
    SignUp = new("sign_up")
  end
end
