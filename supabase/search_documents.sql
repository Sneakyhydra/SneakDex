-- Run this in the Supabase SQL editor.
-- Improves keyword search: indexes title+url+body, prefix/name matching, trigram similarity.

create extension if not exists pg_trgm;

create or replace function public.documents_fill_tsvector()
returns trigger
language plpgsql
as $$
declare
  indexed_text text;
begin
  indexed_text := concat_ws(
    ' ',
    new.title,
    new.url,
    coalesce(new._tmp_content, '')
  );

  begin
    new.content := to_tsvector(
      coalesce(nullif(new.lang, ''), 'simple')::regconfig,
      indexed_text
    );
  exception when others then
    new.content := to_tsvector('simple', indexed_text);
  end;

  return new;
end;
$$;

drop trigger if exists trg_documents_tsvector on public.documents;
create trigger trg_documents_tsvector
before insert or update on public.documents
for each row
execute function public.documents_fill_tsvector();

create index if not exists documents_content_idx
  on public.documents using gin (content);
create index if not exists documents_title_trgm_idx
  on public.documents using gin (title gin_trgm_ops);
create index if not exists documents_url_trgm_idx
  on public.documents using gin (url gin_trgm_ops);

-- Rebuild tsvectors so existing rows include title and URL.
update public.documents set title = title;

create or replace function public.search_documents(q text, limit_count int)
returns table (id uuid, rank real, url text, title text)
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  q_trim text := btrim(coalesce(q, ''));
  tsq tsquery;
  prefix_parts text[] := '{}';
  tok text;
  cleaned text;
begin
  if q_trim = '' then
    return;
  end if;

  foreach tok in array regexp_split_to_array(lower(q_trim), '\s+') loop
    cleaned := regexp_replace(tok, '[^[:alnum:]]', '', 'g');
    if length(cleaned) >= 2 then
      prefix_parts := prefix_parts || (cleaned || ':*');
    end if;
    -- dhruvi -> dhruv:* so short name variants still match
    if length(cleaned) >= 5 then
      prefix_parts := prefix_parts || (left(cleaned, length(cleaned) - 1) || ':*');
    end if;
  end loop;

  begin
    if coalesce(array_length(prefix_parts, 1), 0) = 0 then
      tsq := plainto_tsquery('simple', q_trim);
    else
      tsq := to_tsquery('simple', array_to_string(prefix_parts, ' | '));
    end if;
  exception when others then
    tsq := plainto_tsquery('simple', q_trim);
  end;

  return query
  select
    d.id,
    (
      coalesce(ts_rank_cd(d.content, tsq), 0) * 2
      + coalesce(word_similarity(lower(q_trim), lower(coalesce(d.title, ''))), 0)
      + coalesce(word_similarity(lower(q_trim), lower(coalesce(d.url, ''))), 0)
      + case
          when coalesce(d.title, '') ilike '%' || q_trim || '%' then 0.5
          else 0
        end
    )::real as rank,
    d.url,
    d.title
  from public.documents d
  where
    (tsq is not null and d.content @@ tsq)
    or coalesce(d.title, '') ilike '%' || q_trim || '%'
    or coalesce(d.url, '') ilike '%' || q_trim || '%'
    or word_similarity(lower(q_trim), lower(coalesce(d.title, ''))) > 0.35
    or word_similarity(lower(q_trim), lower(coalesce(d.url, ''))) > 0.35
  order by 2 desc
  limit greatest(coalesce(limit_count, 50), 1);
end;
$$;

grant execute on function public.search_documents(text, int)
  to anon, authenticated, service_role;

-- Score specific Qdrant hit IDs so pgScore is filled on the results
-- the user actually sees, not only on a disjoint keyword top-k.
create or replace function public.rank_documents(q text, doc_ids uuid[])
returns table (id uuid, rank real, url text, title text)
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  q_trim text := btrim(coalesce(q, ''));
  tsq tsquery;
  prefix_parts text[] := '{}';
  tok text;
  cleaned text;
begin
  if q_trim = '' or doc_ids is null or array_length(doc_ids, 1) is null then
    return;
  end if;

  foreach tok in array regexp_split_to_array(lower(q_trim), '\s+') loop
    cleaned := regexp_replace(tok, '[^[:alnum:]]', '', 'g');
    if length(cleaned) >= 2 then
      prefix_parts := prefix_parts || (cleaned || ':*');
    end if;
    if length(cleaned) >= 5 then
      prefix_parts := prefix_parts || (left(cleaned, length(cleaned) - 1) || ':*');
    end if;
  end loop;

  begin
    if coalesce(array_length(prefix_parts, 1), 0) = 0 then
      tsq := plainto_tsquery('simple', q_trim);
    else
      tsq := to_tsquery('simple', array_to_string(prefix_parts, ' | '));
    end if;
  exception when others then
    tsq := plainto_tsquery('simple', q_trim);
  end;

  return query
  select
    d.id,
    (
      coalesce(ts_rank_cd(d.content, tsq), 0) * 2
      + coalesce(word_similarity(lower(q_trim), lower(coalesce(d.title, ''))), 0)
      + coalesce(word_similarity(lower(q_trim), lower(coalesce(d.url, ''))), 0)
      + case
          when coalesce(d.title, '') ilike '%' || q_trim || '%' then 0.5
          else 0
        end
    )::real as rank,
    d.url,
    d.title
  from public.documents d
  where d.id = any(doc_ids)
    and (
      (tsq is not null and d.content @@ tsq)
      or coalesce(d.title, '') ilike '%' || q_trim || '%'
      or coalesce(d.url, '') ilike '%' || q_trim || '%'
      or word_similarity(lower(q_trim), lower(coalesce(d.title, ''))) > 0.35
      or word_similarity(lower(q_trim), lower(coalesce(d.url, ''))) > 0.35
    );
end;
$$;

grant execute on function public.rank_documents(text, uuid[])
  to anon, authenticated, service_role;
grant execute on function public.get_estimated_count()
  to anon, authenticated, service_role;
