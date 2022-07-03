# typed: strict
# frozen_string_literal: true

class OrderProperty
  extend T::Sig

  sig { params(order_by: T::Hash[Symbol, String]).returns(OrderProperty) }
  def self.build(order_by)
    new(order_by[:field], order_by[:direction])
  end

  sig { params(field_: T.nilable(String), direction_: T.nilable(String)).void }
  def initialize(field_ = nil, direction_ = nil)
    @field_ = field_
    @direction_ = direction_
  end

  sig { returns(Symbol) }
  def field
    field_&.downcase&.to_sym.presence || :created_at
  end

  sig { returns(Symbol) }
  def direction
    direction_&.downcase&.to_sym.presence || :asc
  end

  private

  sig { returns(T.nilable(String)) }
  attr_reader :field_

  sig { returns(T.nilable(String)) }
  attr_reader :direction_
end
