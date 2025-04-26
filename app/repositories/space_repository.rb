# typed: strict
# frozen_string_literal: true

class SpaceRepository < ApplicationRepository
  sig { params(space_record: SpaceRecord).returns(Space) }
  def to_model(space_record:)
    Space.new(
      database_id: space_record.id,
      identifier: space_record.identifier,
      name: space_record.name,
      plan: Plan.deserialize(space_record.plan),
      joined_at: space_record.joined_at
    )
  end

  sig { params(space_records: SpaceRecord::PrivateCollectionProxy).returns(T::Array[Space]) }
  def to_models(space_records:)
    space_records.map { to_model(space_record: _1) }
  end
end
