# typed: strict
# frozen_string_literal: true

class SpaceMember < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :space, Space
  const :user, User
end
