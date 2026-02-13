# typed: strict
# frozen_string_literal: true

class EmailConfirmationEvent < T::Enum
  enums do
    EmailUpdate = new("email_update")
    PasswordReset = new("password_reset")
    SignUp = new("sign_up")
  end
end
