# typed: strict
# frozen_string_literal: true

class ExportEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :queued_by_entity, SpaceMemberEntity
  const :space_entity, SpaceEntity
end
