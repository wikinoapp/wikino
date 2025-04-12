# typed: strict
# frozen_string_literal: true

class UserEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :atname, String
  const :name, String
  const :description, String
  const :time_zone, String
end
