# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `UserSessionForm::Creation`.
# Please instead update this file by running `bin/tapioca dsl UserSessionForm::Creation`.


class UserSessionForm::Creation
  sig { returns(T.nilable(::String)) }
  def email; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def email=(value); end

  sig { returns(T.nilable(::String)) }
  def password; end

  sig { params(value: T.nilable(::String)).returns(T.nilable(::String)) }
  def password=(value); end
end
