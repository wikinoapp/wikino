# typed: strict
# frozen_string_literal: true

# Note: ランタイムで型チェックをすると以下のようなエラーが発生することがあるため、
#       一旦 `T::Sig::WithoutRuntime` で型を指定している
# > TypeError (Return value: Expected type User, got type User with value #<User id: "019260a2-6e64-2...-06 16:01:50.799485000 +0900">
# >
# > The expected type and received object type have the same name but refer to different constants.
# > Expected type is User with object id 20180, but received type is User with object id 33580.
# >
# > There might be a constant reloading problem in your application.

class Current < ActiveSupport::CurrentAttributes
  extend T::Sig

  attribute :space, :user

  resets do
    Time.zone = nil
  end

  T::Sig::WithoutRuntime.sig { params(user: T.nilable(User)).void }
  def user=(user)
    super
    Time.zone = user&.time_zone
  end

  T::Sig::WithoutRuntime.sig { returns(User) }
  def user!
    user.not_nil!
  end

  T::Sig::WithoutRuntime.sig { returns(Space) }
  def space!
    space.not_nil!
  end
end
