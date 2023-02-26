class PaginationMetaDto {
  pageSize: number;
  page: number;
  total: number;
}

export class PaginationResponseDto<T = any> {
  data: Array<T>;
  meta: PaginationMetaDto;
}
