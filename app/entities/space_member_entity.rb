# typed: strict
# frozen_string_literal: true

class SpaceMemberEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :space_entity, SpaceEntity
  const :user_entity, UserEntity
end
