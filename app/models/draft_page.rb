# typed: strict
# frozen_string_literal: true

class DraftPage < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :modified_at, ActiveSupport::TimeWithZone
  const :space, Space
  const :page, Page
end
