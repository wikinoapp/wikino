# typed: strict
# frozen_string_literal: true

class Export < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :queued_by, SpaceMember
  const :space, Space
end
