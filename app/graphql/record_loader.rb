# typed: true
# frozen_string_literal: true

# https://github.com/Shopify/graphql-batch/blob/8873ffdcbb3aea3a44aa64fa7a5470d678157ba1/examples/record_loader.rb
class RecordLoader < GraphQL::Batch::Loader
  def initialize(model, column: model.primary_key, where: nil)
    super()
    @model = model
    @column = column.to_s
    @column_type = model.type_for_attribute(@column)
    @where = where
  end

  def load(key)
    super(@column_type.cast(key))
  end

  def perform(keys)
    query(keys).each { |record| fulfill(record.public_send(@column), record) }
    keys.each { |key| fulfill(key, nil) unless fulfilled?(key) }
  end

  private

  def query(keys)
    scope = @model
    scope = scope.where(@where) if @where
    scope.where(@column => keys)
  end
end
